# Go 并发编程核心基石与锁安全实践

> 基于 easy-chat 项目 `groupMsgRead` / `msgReadTransfer` 模块的实际代码，系统梳理 Go 并发编程的知识框架与安全实践。

---

## 一、Go 并发编程六大基石全景

```
┌─────────────────────────────────────────────────────────────┐
│                  Go 并发编程六大基石                          │
│                                                             │
│  ① 协程(Goroutine)    ② 通道(Channel)    ③ 同步原语(Sync)  │
│     调度与生命周期        通信与流控          互斥与协调       │
│                                                             │
│  ④ 原子操作(Atomic)   ⑤ 上下文(Context)   ⑥ 内存模型(MM)   │
│     无锁编程            取消与传播          可见性与序        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
                          │
                    ┌─────▼──────┐
                    │ 并发模式层  │  ← 组合六大基石，解决实际问题
                    └────────────┘
```

四层认知结构：

```
┌───────────────────────────────────────┐
│  第四层：并发模式（组合应用）            │  生产消费 · 合并批处理
├───────────────────────────────────────┤  Fan-in/out · Pool · Pipeline
│  第三层：协调层（锁 + 原子 + 上下文）   │  Mutex · Atomic · Context
├───────────────────────────────────────┤
│  第二层：通信层（Channel）             │  CSP 通信 · 流控背压 · 信号广播
├───────────────────────────────────────┤
│  第一层：执行层（Goroutine）           │  GMP 调度 · 生命周期管理
├───────────────────────────────────────┤
│  地基：内存模型（Memory Model）        │  happens-before · 可见性 · 顺序性
└───────────────────────────────────────┘
```

---

## 二、基石①：协程（Goroutine）

### 2.1 GMP 调度模型

```
G - Goroutine  (用户态，初始栈 2KB，可创建数百万)
M - Machine    (操作系统线程，通常等于 CPU 核数)
P - Processor  (逻辑处理器，GOMAXPROCS 控制)

调度流程：G 被 P 调度到 M 上执行
阻塞时(channel/IO)，G 挂起，P 转移给其他 M 继续调度
```

### 2.2 关键特性

| 特性 | 说明 |
|------|------|
| 极轻量 | 初始栈 2KB，按需动态扩缩，可创建百万级 |
| 非抢占 | 基于协作式抢占（函数调用点、channel 操作等） |
| 无 ID | 没有 goroutine ID，无法直接"杀掉"某个协程 |
| 泄漏风险 | 协程持有资源不退出 = 内存泄漏 |

### 2.3 生命周期管理：done channel 模式

```go
// ❌ 协程泄漏：没有退出机制
go func() {
    for {
        doSomething()  // 永远不会停
    }
}()

// ✅ 标准模式：done channel 控制退出
go func() {
    for {
        select {
        case <-done:
            return        // 优雅退出
        case <-ticker.C:
            doSomething()
        }
    }
}()
```

> **easy-chat 体现**：`groupMsgRead.transfer()` 通过 `<-m.done` 接收退出信号，
> `clear()` 中 `close(m.done)` 触发协程退出。

### 2.4 协程泄漏排查

```bash
# 使用 runtime 查看协程数量
runtime.NumGoroutine()

# 使用 pprof 分析
go tool pprof http://localhost:6060/debug/pprof/goroutine

# 使用 race detector
go test -race ./...
```

---

## 三、基石②：通道（Channel）

### 3.1 CSP 哲学

> **Don't communicate by sharing memory; share memory by communicating.**
> 不要通过共享内存来通信，而要通过通信来共享内存。

### 3.2 Channel 分类

| 类型 | 声明 | 语义 | 阻塞行为 |
|------|------|------|---------|
| 无缓冲 | `make(chan T)` | 同步通道 | 发送方阻塞直到接收方就绪 |
| 有缓冲 | `make(chan T, N)` | 异步通道 | 缓冲满前发送不阻塞 |
| 单向读 | `<-chan T` | 只读约束 | 编译期类型检查 |
| 单向写 | `chan<- T` | 只写约束 | 编译期类型检查 |

### 3.3 Channel 三大作用

| 作用 | 说明 | easy-chat 体现 |
|------|------|---------------|
| **通信** | 协程间传递数据 | `m.pushCh <- push` 传递已读推送 |
| **流控** | 背压控制速率 | `make(chan *ws.Push, 1)` 缓冲=1 产生背压 |
| **信号** | 通知退出/状态变更 | `close(m.done)` 广播退出信号 |

### 3.4 缓冲大小的设计原则

缓冲大小不是"越大越顺畅"，而是要和业务逻辑的节奏对齐。

```
缓冲=0 (无缓冲)
  优点：同步语义强，零堆积
  缺点：生产者每次必须等消费者就绪，吞吐受限
  适用：强同步场景、必须确认对方收到

缓冲=1 (推荐默认)
  优点：最小缓冲避免生产者卡死，背压倒逼合并效率最大化
  缺点：消费者慢时生产者仍可能短暂阻塞
  适用：生产消费速率接近、有合并/批处理需求

缓冲=N (谨慎使用)
  优点：生产者几乎不阻塞
  缺点：消息堆积延迟、空闲通知被卡、合并窗口被压缩
  适用：消费者明确比生产者快、消息允许短暂延迟
```

**easy-chat 中缓冲=1 的设计分析**：

```go
push: make(chan *ws.Push, 1)
```

- `Consume()` 是 kq 消费者回调，在热路径上，不能长时间阻塞
- `transfer()` 是独立协程，调用 `Transfer()` 做网络推送，可能耗时
- 缓冲=1：允许一条消息排队，不会卡住 `Consume()`
- 缓冲过大：群聊空闲通知（`idlePush`）会被延迟，`groupMsgRead` 无法及时释放

### 3.5 select 四种用法

```go
// 1. 多路复用
select {
case <-ticker.C:
    // 定时触发
case <-done:
    // 退出信号
}

// 2. 非阻塞发送/接收
select {
case ch <- value:
    // 发送成功
default:
    // 满了就跳过
}

// 3. 超时控制
select {
case result := <-ch:
    // 收到结果
case <-time.After(3 * time.Second):
    // 超时
}

// 4. close-once 模式（防止重复 close 导致 panic）
select {
case <-done:
    // 已关闭，不做任何事
default:
    close(done)
}
```

### 3.6 close 的广播语义

```go
// close(ch) 后，所有 <-ch 的接收者都会立即收到零值
// 这使得 close 天然适合做"广播退出信号"

close(m.done)  // 一次调用，所有监听 <-m.done 的协程全部收到
```

> **注意**：`close` 只能调用一次，重复 `close` 会 panic。
> easy-chat 中的 `clear()` 用 select+default 实现了 close-once 保护。

---

## 四、基石③：同步原语（Sync）

### 4.1 完整工具箱

```
sync 包
├── Mutex          互斥锁          ← 读写都互斥
├── RWMutex        读写锁          ← 读多写少时优于 Mutex
├── WaitGroup      等待一组协程完成  ← 批量并发任务收拢
├── Once           只执行一次       ← 单例初始化
├── Cond           条件变量         ← 等待/通知模式
├── Map            并发安全 map     ← 读多写少的特殊 map
├── Pool           对象池          ← 减少GC压力
└── SingleFlight   合并相同请求     ← 防缓存击穿

扩展包
├── sync/atomic    原子操作        ← 单变量无锁编程
└── errgroup       带错误收集的 WaitGroup
```

### 4.2 Mutex vs RWMutex 选择

```go
// 用 RWMutex 的条件：读远多于写，且读操作耗时长
// 如果读写频率接近，RWMutex 的额外开销反而得不偿失

// groupMsgRead 中为什么用 Mutex 而不是 RWMutex？
// 因为 isIdle() 虽然是"读"，但紧跟的 clear()/mergePush() 是"写"
// 读写交替频繁，用 Mutex 更简单高效

type groupMsgRead struct {
    mu sync.Mutex  // ← 读写均衡场景
}
```

### 4.3 WaitGroup 标准模式

```go
var wg sync.WaitGroup

for i := 0; i < 10; i++ {
    wg.Add(1)        // ← 必须在 goroutine 外面 Add
    go func() {
        defer wg.Done()
        doWork()
    }()
}
wg.Wait()            // ← 等待所有完成
```

### 4.4 sync.Map 适用场景

```go
// sync.Map 的两个特殊场景：
// 1. key 一旦写入很少修改（如缓存）
// 2. 多个 goroutine 读写不同的 key（无竞争）

// 不适用：频繁读写相同 key → 性能不如 map + Mutex
```

---

## 五、基石④：原子操作（Atomic）

### 5.1 适用场景：单变量的原子读写

```go
// Go 1.19+ 推荐使用类型化的 atomic

var count atomic.Int64
count.Add(1)                     // 原子自增
v := count.Load()                // 原子读取

var state atomic.Pointer[Data]   // 原子指针
state.Store(&data)
p := state.Load()

var flag atomic.Bool             // 原子布尔
flag.Store(true)
if flag.Load() { ... }
```

### 5.2 Atomic vs Mutex 边界

```
                    操作复杂度
                    低 ◄──────────────────► 高

            atomic     │        Mutex
单变量读改写 ────┤───────────────┤
(计数器/标志位)   │               │
                  │    多字段组合操作
                  │    (如 isIdle 读3个字段)
                  │               │
            ──────┴───────────────┴─────
            简单                 复杂
```

> **isIdle()** 涉及 3 个字段（`pushTime`、`push`、`count`）的组合判断，
> 无法用 atomic 解决，必须用 Mutex。

---

## 六、基石⑤：上下文（Context）

### 6.1 四种用法

```go
// 1. 取消传播（最核心）
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
go func() {
    select {
    case <-ctx.Done():
        return   // 父调用 cancel()，所有子协程收到信号
    }
}()

// 2. 超时自动取消
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// 3. 截止时间
ctx, cancel := context.WithDeadline(ctx, time.Now().Add(3*time.Second))
defer cancel()

// 4. 值传递（谨慎使用，仅用于请求域的元数据）
ctx := context.WithValue(ctx, "userId", "123")
userId := ctx.Value("userId").(string)
```

### 6.2 Context 使用原则

| 原则 | 说明 |
|------|------|
| 不要传递 nil | 用 `context.TODO()` 代替 |
| 不要传递可选参数 | `WithValue` 仅用于跨 API 边界的元数据 |
| 作为第一个参数 | `func DoSomething(ctx context.Context, ...)` |
| 不要持有 context | 长生命周期对象不要保存 context 引用 |

### 6.3 easy-chat 中的体现

```go
// msgReadTransfer.go
m.Transfer(context.Background(), push)
// 用 Background() 而非 Consume 传入的 ctx
// 因为 push 是异步的，原始 ctx 可能已失效
// 改进方向：如需超时控制，应创建新的 WithTimeout context
```

---

## 七、基石⑥：内存模型（Memory Model）— 并发安全的底座

> 内存模型是所有并发机制的**地基**。不理解 happens-before，写出的并发代码就像在沙上盖楼——
> 偶尔能跑，但随时可能因为编译器重排或 CPU 缓存而崩塌。

### 7.1 为什么需要内存模型？

#### 现实：编译器和 CPU 会"偷偷改顺序"

你以为代码是按写的顺序执行的？并不是。

```go
// 你写的代码
X = 1      // 语句 A
flag = true // 语句 B

// 编译器/CPU 可能重排为：
flag = true // 语句 B 先执行了！
X = 1      // 语句 A 后执行
```

**单协程下没问题**：重排不影响最终结果（A 和 B 没有依赖关系）。

**多协程下出问题**：另一个协程看到 `flag == true` 后去读 `X`，
结果 `X` 还是 0——因为 A 被排到了 B 后面。

```
协程1                   协程2
──────                  ──────
X = 1                   while !flag {}
flag = true    ──►      print(X)  // 期望 1，实际可能 0！

原因：编译器/CPU 可能重排协程1的两条语句
```

这就是内存模型要解决的问题：**在多协程环境下，什么操作结果对另一个协程一定可见？**

### 7.2 happens-before：可见性的传递链

#### 定义

```
如果事件 A happens-before 事件 B，
那么 B 一定能观察到 A 产生的所有内存写操作结果。
```

**注意**：happens-before 不是"时间上先发生"，而是"保证可见"。

- 时间上 A 先发生，但如果没有 happens-before 关系，B 可能看不到 A 的结果
- happens-before 是逻辑上的保证，不是时间上的先后

#### happens-before 的八大来源

```
┌──────────────────────────────────────────────────────────────┐
│              happens-before 八大来源                          │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ① 单协程顺序：同一协程内，语句按代码顺序 happens-before      │
│                                                              │
│  ② channel 发送→接收：ch <- x  happens-before  <-ch          │
│                                                              │
│  ③ channel close→接收：close(ch) happens-before <-ch 收零值  │
│                                                              │
│  ④ Mutex Unlock→Lock：Unlock happens-before 下一次 Lock      │
│                                                              │
│  ⑤ Once f()→Wait：f() 返回 happens-before Wait() 返回       │
│                                                              │
│  ⑥ WaitGroup Done→Wait：Done happens-before Wait 返回        │
│                                                              │
│  ⑦ atomic 操作：Load 可见 Store 的值（Go 1.19+ 明确）         │
│                                                              │
│  ⑧ 传递性：A hb B 且 B hb C → A hb C（最重要的推理工具）     │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### 7.3 八大来源逐条详解

#### ① 单协程顺序（Sequential Consistency within Goroutine）

```go
// 同一协程内，代码顺序就是 happens-before 顺序
X = 1       // A
Y = X + 1   // B —— A happens-before B（因为 B 依赖 A 的结果）
```

**局限**：这个保证只在同一协程内有效！其他协程看不到这个顺序。

```go
// 协程1
X = 1       // A
flag = true // B    A hb B（同一协程内）

// 协程2
if flag {    // C
    print(X) // D    C hb D（同一协程内）
}
// 但 A 不 happens-before D！
// 因为 A→B 和 C→D 是两条独立的链，没有交叉
```

#### ② Channel 发送→接收

```go
// 协程1
ch <- value  // A

// 协程2
value = <-ch // B

// A happens-before B
// 所以协程1在发送前所有的写操作，对协程2都可见
```

**传递性推导**：

```go
// 协程1
X = 1           // ① 单协程内 hb
ch <- signal     // ② 发送 hb 接收

// 协程2
<-ch             // ③ 接收
print(X)         // ④ 单协程内 hb

// 推理链：X=1 hb ch<- hb <-ch hb print(X)
// 由传递性：X=1 hb print(X) → 一定打印 1
```

> **easy-chat 体现**：`Consume()` 通过 `m.push <- push` 发送推送消息，
> `transfer()` 通过 `<-m.push` 接收，接收方能看到发送方的所有写操作结果。

#### ③ Channel close→接收

```go
// 协程1
ch <- lastValue
close(ch)        // A

// 协程2
for v := range ch {  // B
    // 先收到 lastValue，再收到零值（channel 已关闭）
}

// A happens-before B 收到零值
```

> **easy-chat 体现**：`clear()` 中 `close(m.done)` 后，
> `groupMsgRead.transfer()` 中的 `<-m.done` 一定能感知到 close 之前的所有状态变更。

#### ④ Mutex Unlock→Lock（最常用）

```go
// 协程1
mu.Lock()
X = 1
mu.Unlock()     // A —— Unlock

// 协程2
mu.Lock()       // B —— Lock
print(X)        // 一定打印 1
mu.Unlock()

// A happens-before B（Unlock hb 下一次 Lock）
```

**完整推理链**：

```
协程1:  Lock → 写X=1 → Unlock(A)
                                  │
                          happens-before
                          （Mutex 规则）
                                  │
                                  ▼
协程2:            Lock(B) → 读X → Unlock

传递性：写X=1 hb 读X → X 一定可见
```

> **easy-chat 体现**：`mergePush()` 在 `m.mu.Lock()` 下写 `count`/`push`，
> `isIdle()` 在 `m.mu.Lock()` 下读 `count`/`push`，
> 前者的 Unlock happens-before 后者的 Lock，所以读到的值一定是最新的。

#### ⑤ Once f()→Wait

```go
var once sync.Once
var data int

once.Do(func() {
    data = 42    // A —— f() 内的写
})               // f() 返回

// 其他协程
print(data)     // B —— 一定打印 42
// 因为 A happens-before B
```

#### ⑥ WaitGroup Done→Wait

```go
var wg sync.WaitGroup
var result int

wg.Add(1)
go func() {
    result = compute()  // A
    wg.Done()            // B
}()

wg.Wait()               // C
print(result)           // D —— 一定打印 compute() 的结果

// 推理链：A hb B（单协程），B hb C（WaitGroup 规则），C hb D（单协程）
// 传递性：A hb D → result 一定可见
```

#### ⑦ atomic 操作

```go
var flag atomic.Bool
var data string

// 协程1
data = "hello"       // A
flag.Store(true)     // B —— A hb B（单协程）

// 协程2
if flag.Load() {     // C
    print(data)      // D —— C hb D（单协程）
}

// atomic 保证：B 的 Store 对 C 的 Load 可见
// 推理链：A hb B hb C hb D → A hb D → data 一定可见
```

> **注意**：atomic 保证了 Store/Load 本身的可见性，但搭配其他变量的读写时，
> 需要靠单协程内顺序 + 传递性来建立完整的 happens-before 链。

#### ⑧ 传递性（最核心的推理工具）

```
A happens-before B
B happens-before C
──────────────────
A happens-before C    ← 传递性
```

传递性是连接不同同步机制的"胶水"。几乎所有复杂的并发推理，都要靠传递性把多条短链拼成长链。

**实战推理示例**：

```
协程1:  data = 42   →  mu.Unlock()
                              │ hb (Mutex规则)
                              ▼
协程2:         mu.Lock()  →  ch <- signal
                                       │ hb (Channel规则)
                                       ▼
协程3:                        <-ch  →  print(data) = 42

推理链：data=42 hb Unlock hb Lock hb ch<- hb <-ch hb print
传递性：data=42 hb print → 一定打印 42
```

### 7.4 用 happens-before 分析 easy-chat 的并发安全

#### 分析1：mergePush 写 → isIdle 读（Mutex 保证）

```go
// mergePush (协程A)
m.mu.Lock()
m.count++                    // 写①
m.push.ReadRecords[k] = v   // 写②
m.mu.Unlock()  ──────────────────┐
                                  │ happens-before
                                  │ (Mutex 规则)
                                  ▼
// isIdle (协程B)
             m.mu.Lock()
             读 m.count  ──────────── 看到写①
             读 m.push   ──────────── 看到写②
             m.mu.Unlock()
```

#### 分析2：Consume 写推送 → transfer 读推送（Channel 保证）

```go
// Consume (协程A)
push.ReadRecords = records   // 写①
m.push <- push              // 发送  ───────────┐
                                                   │ happens-before
                                                   │ (Channel 规则)
                                                   ▼
// transfer (协程B)
                              push := <-m.push    // 接收
                              读 push.ReadRecords  ── 看到写①
```

#### 分析3：clear 关闭 → transfer 退出（close + Mutex 保证）

```go
// clear (协程A)
m.mu.Lock()       // 在 MsgReadTransfer.transfer() 中
m.groupMses[id].clear()
  close(m.done)   // 信号①  ──────────────────────┐
  m.push = nil    // 写②                             │
delete(m.groupMses, id)                             │ happens-before
m.mu.Unlock()  ────────────────────────────┐       │ (close 规则)
                                            │       │
                                            │ hb    │
                                            │       ▼
// groupMsgRead.transfer (协程B)            ▼
                              <-m.done      // 收到信号①
                              return        // 退出
```

### 7.5 反面教材：没有 happens-before 会怎样

#### 反例1：双重检查锁（Double-Check Locking）的陷阱

```go
// 错误的双重检查
var instance *Config
var mu sync.Mutex

func GetInstance() *Config {
    if instance == nil {         // 第一次检查，无锁！
        mu.Lock()
        if instance == nil {     // 第二次检查，有锁
            instance = &Config{...}  // 写
        }
        mu.Unlock()
    }
    return instance  // 可能返回未初始化完全的对象
}

// 问题：第一次检查时没有 happens-before 关系
// 可能读到 CPU 缓存中的旧值(nil)，或者半初始化的对象
```

```go
// 正确做法：使用 sync.Once
var once sync.Once
var instance *Config

func GetInstance() *Config {
    once.Do(func() {
        instance = &Config{...}
    })
    return instance  // Once 保证 happens-before，一定可见
}
```

#### 反例2：无同步的"标志位"通信

```go
// 错误：用普通变量做协程间信号
var done bool
var data string

// 协程1
data = "hello"
done = true        // 没有 happens-before 保证！

// 协程2
for !done {}       // 可能永远看不到 done=true（CPU 缓存）
print(data)
```

```go
// 正确做法1：用 channel 做信号
var data string
ch := make(chan struct{})

// 协程1
data = "hello"
close(ch)           // close hb <-ch（Channel 规则）

// 协程2
<-ch                // 一定能收到
print(data)         // 一定能看到 "hello"
```

```go
// 正确做法2：用 atomic 做信号
var done atomic.Bool
var data string

// 协程1
data = "hello"
done.Store(true)    // atomic 可见性保证

// 协程2
for !done.Load() {} // 一定能看到 true
print(data)         // 一定能看到 "hello"
```

#### 反例3：无锁的 map 并发读写

```go
// 错误：Go 原生 map 并发读写会 panic
var m = make(map[string]int)

// 协程1
m["key"] = 1       // 写

// 协程2
_ = m["key"]       // 读 → panic: concurrent map read and map write

// 原因：map 内部没有任何同步机制，并发访问是未定义行为
```

### 7.6 happens-before 推理实战五步法

面对任何并发代码，都可以用以下五步判断是否安全：

```
步骤1：找出所有共享变量（被多个协程读写的变量）
         │
步骤2：对每个共享变量，画出"写→读"的协程交叉点
         │
步骤3：检查每个交叉点是否有 happens-before 链
         │   ├── 有 → 安全
         │   └── 没有 → 数据竞争
         │
步骤4：对有 happens-before 链的，用传递性验证完整性
         │   ├── 完整 → 安全
         │   └── 断裂 → 部分可见，可能不安全
         │
步骤5：用 go test -race 验证
```

**用五步法分析 `isIdle()` 的安全性**：

```
步骤1：共享变量 = count, push, pushTime

步骤2：写→读交叉点：
  mergePush 写 count/push → isIdle 读 count/push/pushTime
  transfer 写 count/push/pushTime → isIdle 读 count/push/pushTime

步骤3：检查 happens-before 链：
  mergePush: Lock → 写 → Unlock ──hb──► Lock(isIdle) → 读 → Unlock (有)
  transfer: Lock → 写 → Unlock ──hb──► Lock(isIdle) → 读 → Unlock (有)

步骤4：传递性验证：
  写 → Unlock → Lock → 读，链完整

步骤5：结论 — 安全，所有共享变量的读写都有 Mutex 保护的 happens-before 链
```

### 7.7 常见误解澄清

| 误解 | 事实 |
|------|------|
| "先写的代码，后读一定能看到" | 不对！跨协程没有同步就没有 happens-before，可能看不到 |
| "volatile/atomic 能解决一切" | atomic 只保证单变量的可见性，多字段组合仍需锁 |
| "加锁只是为了防止并发写" | 加锁也保证了读的可见性（Unlock hb Lock） |
| "channel 不需要锁" | channel 本身是安全的，但 channel 前后的共享变量操作仍需同步 |
| "我的代码跑得好好的" | data race 是未定义行为，99.9% 正常不等于正确，0.1% 会出致命 bug |
| "x86 不会重排" | x86 确实是强序模型，但编译器仍会重排；ARM/ARM64 更是弱序 |
| "race detector 没报错就安全" | race detector 只能发现运行时实际触发的竞争，不能证明代码无竞争 |

### 7.8 一句话总结

> **happens-before 是并发安全的"法律"：有它就有保证，没它就是未定义。**
> 写并发代码时，对每一个跨协程的"写→读"交叉点，都要画出 happens-before 链。
> 画不出来，就是数据竞争。

---

## 八、锁安全实践

### 8.1 什么时候必须加锁

```
                 另一个协程会写吗？
                      │
            ┌──No─────┴─────Yes────┐
            │                       │
       不需要加锁              需要同步机制
       (只读共享)                   │
                          ┌────────┴────────┐
                          │                  │
                     只读多写少           读写均衡
                          │                  │
                    sync.RWMutex         sync.Mutex
                    (读读不互斥)         (读写都互斥)
```

### 8.2 必须加锁的五大场景

| 场景 | 原因 | easy-chat 体现 |
|------|------|---------------|
| 读写共享变量 | data race，未定义行为 | `m.count`、`m.push`、`m.pushTime` |
| 多字段组合判断 | 快照不一致（torn read） | `isIdle()` 同时读 3 个字段 |
| 检查后操作（check-then-act） | TOCTOU 竞态 | `isIdle()` → `clear()` → `delete()` |
| 共享 map 并发访问 | Go map 非并发安全，会 panic | `m.groupMses` 用 `m.mu` 保护 |
| channel + 状态联动 | channel 操作和状态修改需原子 | `close(m.done)` 配合 `m.push = nil` |

### 8.3 "只是判断"也需要加锁的原因

以 `isIdle()` 为例：

```go
func (m *groupMsgRead) isIdle() bool {
    pushTime := m.pushTime          // 读取 ①
    val := GroupMsgReadRecordDelayTime*2 - time.Since(pushTime)
    if val <= 0 && m.push == nil && m.count == 0 {  // 读取 ② ③
        return true
    }
    return false
}
```

不加锁的三种问题：

**问题1：数据竞争（Data Race）**

```go
// 协程A：isIdle 读取        协程B：transfer 写入
读 m.push == nil            m.push = nil
读 m.count == 0             m.count = 0
// 对同一变量的并发读写，没有同步 = data race
```

**问题2：快照不一致（Torn Read）**

```go
时间线：
  协程B(transfer)              协程A(isIdle)
  ──────────────              ──────────────
  m.count = 0
  m.push = nil
                               读 m.push → nil ✅
                               读 m.count → 0 ✅
  mergePush() 被调用
    m.count++  → 1
    合并 ReadRecords
                               读 m.pushTime → 很久前 ✅
                               → 判定 isIdle = true ❌ 错误！
                                  实际已有新消息进来了
```

**问题3：导致严重后果**

```go
// isIdle() 返回了错误的 true →
m.groupMses[id].clear()       // 误杀协程
delete(m.groupMses, id)        // 从 map 删除
// 正在合并的消息丢失，已读回执丢失
```

### 8.4 大小写分离模式

```go
// 对外暴露（加锁）— 供外部调用
func (m *groupMsgRead) IsIdle() bool {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.isIdle()
}

// 内部使用（不加锁）— 仅在已持锁的上下文中调用
func (m *groupMsgRead) isIdle() bool {
    pushTime := m.pushTime
    val := GroupMsgReadRecordDelayTime*2 - time.Since(pushTime)
    return val <= 0 && m.push == nil && m.count == 0
}
```

要点：
- 大写 `IsIdle()`：公开方法，自己加锁，调用者无需关心锁
- 小写 `isIdle()`：内部方法，不加锁，仅在已持锁的上下文中调用
- Go 的 `sync.Mutex` **不可重入**，如果内部也调用 `IsIdle()` 会死锁

### 8.5 加锁的常见陷阱

| 陷阱 | 说明 | 正确做法 |
|------|------|---------|
| 忘记加锁 | "只是读一下" | 只要读的字段可能被其他协程写，就必须加锁 |
| 锁粒度过粗 | 整个函数一把锁 | 缩小临界区，只锁必须的部分 |
| 锁粒度过细 | 多把锁分步操作 | 多字段组合操作必须在同一把锁下 |
| 忘记 Unlock | 异常路径跳过 Unlock | 始终用 `defer m.mu.Unlock()` |
| 重入死锁 | Mutex 不可重入 | 大小写分离，内部方法不加锁 |
| 重复 close | close 已关闭的 channel 会 panic | select+default 的 close-once 模式 |
| 锁与 channel 交叉 | 持锁期间操作 channel 可能死锁 | 先释放锁，再操作 channel |

### 8.6 持锁期间的操作原则

```go
// ✅ 正确：先释放锁，再操作 channel
m.mu.Lock()
if m.count >= threshold {
    push := m.push
    m.count = 0
    m.push = nil
    m.mu.Unlock()           // 先释放锁
    m.pushCh <- push         // 再操作 channel（可能阻塞）
    continue
}
m.mu.Unlock()

// ❌ 危险：持锁操作 channel
m.mu.Lock()
m.pushCh <- m.push          // 如果 channel 满了，阻塞等待
m.mu.Unlock()               // 期间锁一直被持有，其他协程全部阻塞
```

> **easy-chat 体现**：`groupMsgRead.transfer()` 中，先 `Unlock` 再 `m.pushCh <- push`。

---

## 九、六大并发模式

### 模式1：生产者-消费者

```
生产者 ──► channel ──► 消费者

easy-chat 体现：
Consume() ──► push channel(缓冲1) ──► transfer()
```

### 模式2：Fan-out / Fan-in

```
Fan-out：一个生产者，多个消费者（分发）
Fan-in：多个生产者，一个消费者（汇聚）

easy-chat 体现（Fan-in）：
多个 groupMsgRead.transfer() ──► 同一个 push channel ──► MsgReadTransfer.transfer()
```

### 模式3：Pipeline（流水线）

```
Stage1 ──► Stage2 ──► Stage3

适用：数据需要经过多步处理，每步可以并行
```

### 模式4：Worker Pool（工作池）

```
         ┌─ worker1 ─┐
task ───►├─ worker2 ─┤───► result
         └─ worker3 ─┘

适用：限制并发数，避免资源耗尽
```

### 模式5：合并批处理

```
多个消息 ──► 合并到同一对象 ──► 累积 ──► 达量/超时 ──► 一次性推送

easy-chat 体现：
Consume() ──► mergePush() ──► 累积 ──► 10条/1秒 ──► pushCh

核心思想：用时间换空间，用背压换合并效率
```

### 模式6：通知退出（done channel）

```
close(done) 广播退出信号
     │
     ├── goroutine1: <-done → return
     ├── goroutine2: <-done → return
     └── goroutine3: <-done → return

easy-chat 体现：
clear() → close(m.done) → groupMsgRead.transfer() 退出
```

---

## 十、easy-chat 并发设计全景回顾

```
                    Kafka 消息
                        │
                        ▼
                ┌───────────────┐
                │   Consume()   │  kq 消费者回调（热路径）
                └───────┬───────┘
                        │
            ┌───────────┼───────────┐
            │                       │
      私聊(SingleChat)        群聊(GroupChat)
            │                       │
            │               ┌───────┴───────┐
            │               │               │
            │         mergePush()      newGroupMsgRead()
            │         合并到已有对象      创建新合并器
            │               │               │
            │               │         go m.transfer()
            │               │         ticker 周期检查
            │               │               │
            │               │     ┌─────────┼─────────┐
            │               │   达量推送   超时推送    空闲
            │               │     │         │          │
            ▼               ▼     ▼         ▼          │
        ┌─────────────────────────────────────┐        │
        │    push channel (缓冲=1)             │        │
        │    作用：通信 + 流控 + 背压           │◄───────┘
        └─────────────────┬───────────────────┘   空闲通知
                          │
                          ▼
                ┌─────────────────┐
                │   transfer()    │  单协程 for-range 消费
                │   ├ Transfer()  │  网络推送
                │   └ 清理空闲     │  clear() → close(done)
                └─────────────────┘

并发安全机制：
  ├ push channel   → 协程间通信（CSP）
  ├ mu Mutex       → 多字段组合读写保护
  ├ m.done channel → 协程退出信号（close 广播）
  ├ close-once     → select+default 防重复 close
  └ 缓冲=1 背压    → 倒逼合并效率最大化
```

---

## 十一、排查工具

| 工具 | 用途 | 命令 |
|------|------|------|
| Race Detector | 检测数据竞争 | `go test -race ./...` |
| pprof | 分析协程泄漏、CPU/内存 | `go tool pprof http://localhost:6060/debug/pprof/goroutine` |
| go vet | 静态分析锁使用问题 | `go vet ./...` |
| staticcheck | 高级静态检查 | `staticcheck ./...` |
| NumGoroutine | 运行时监控协程数 | `runtime.NumGoroutine()` |

---

## 十二、速查清单

### 什么时候加锁？

- [x] 读写被多个协程访问的共享变量
- [x] 多字段组合判断（如 `isIdle()` 读 3 个字段）
- [x] check-then-act 操作序列
- [x] 并发访问 Go 原生 map（非 `sync.Map`）
- [x] channel 操作与状态修改需要联动时

### 什么时候不需要加锁？

- [ ] 只读的常量和配置
- [ ] 协程局部变量（无共享）
- [ ] channel-only 通信（channel 本身是同步的）
- [ ] 使用了 `sync/atomic` 的单变量操作
- [ ] 只有一个协程访问的数据

### 缓冲大小选择？

- [ ] 需要严格同步 → 缓冲=0
- [x] 通用场景、有合并需求 → 缓冲=1
- [ ] 允许短暂延迟、消费速度明确更快 → 缓冲=N（谨慎）

### 协程退出？

- [x] done channel 模式（`close(done)` 广播退出）
- [ ] context 取消传播
- [x] close-once 保护（`select { case <-done: default: close(done) }`）
