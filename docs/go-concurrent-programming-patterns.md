# Go 高并发编程核心编码方式全景指南

> 透过现象看本质：掌握 `go func() + for{}` 和 `sync.WaitGroup` 的底层逻辑与组合模式

---

## 一、Go 并发编程的本质

### 1.1 两种核心编码范式

你说的完全正确，Go 高并发编程确实围绕两种核心写法展开：

```go
// 范式1：go func() + for{} —— 长生命周期协程（持续工作）
go func() {
    for {
        // 持续监听/处理/循环
    }
}()

// 范式2：sync.WaitGroup —— 短生命周期协程（批量任务）
var wg sync.WaitGroup
for i := 0; i < n; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        // 完成一个独立任务
    }()
}
wg.Wait()
```

### 1.2 本质区别

| 维度 | go + for{} | WaitGroup |
|------|-----------|-----------|
| **生命周期** | 长期运行（服务、监听器） | 短期任务（批处理、并行计算） |
| **退出方式** | 主动退出（done channel / context） | 自然退出（任务完成） |
| **使用场景** | 事件驱动、持续消费 | 任务分发、并行加速 |
| **阻塞点** | select / channel 接收 | wg.Wait() 等待 |
| **典型模式** | 生产者-消费者、事件循环 | Fan-out/Fan-in、Worker Pool |

### 1.3 底层统一：都是 Goroutine + 同步原语

```
所有并发模式的底层公式：

Goroutine（执行单元） + 同步机制（Channel / Mutex / WaitGroup / Atomic） = 并发模式
```

---

## 二、范式1：go func() + for{} 的 6 种核心编码方式

### 2.1 事件循环模式（Event Loop）

**本质**：通过 `select` 监听多个 channel，处理事件驱动

```go
// ✅ 标准模式：监听多个事件源
func EventLoop() {
    done := make(chan struct{})
    events := make(chan Event, 100)
    ticker := time.NewTicker(time.Second)
    
    go func() {
        for {
            select {
            case <-done:
                fmt.Println("退出事件循环")
                return
            
            case event := <-events:
                handleEvent(event)
            
            case <-ticker.C:
                handleTick()
            }
        }
    }()
}

// 退出方式
close(done)  // 广播退出信号
```

**实际案例**：WebSocket 连接管理

```go
type Client struct {
    conn   *websocket.Conn
    send   chan []byte
    done   chan struct{}
}

func (c *Client) ReadLoop() {
    defer close(c.done)
    
    for {
        select {
        case <-c.done:
            return
        default:
            _, message, err := c.conn.ReadMessage()
            if err != nil {
                return
            }
            c.handleMessage(message)
        }
    }
}

func (c *Client) WriteLoop() {
    defer close(c.done)
    
    for {
        select {
        case <-c.done:
            return
        case message := <-c.send:
            c.conn.WriteMessage(websocket.TextMessage, message)
        }
    }
}
```

### 2.2 生产者-消费者模式

**本质**：通过 channel 解耦生产和消费，实现背压控制

```go
// ✅ 经典生产者-消费者
func ProducerConsumer() {
    ch := make(chan int, 10)  // 缓冲 channel
    done := make(chan struct{})
    
    // 生产者
    go func() {
        defer close(ch)  // 生产完成，关闭 channel
        for i := 0; i < 100; i++ {
            ch <- i  // 如果 buffer 满了，会阻塞（背压）
        }
    }()
    
    // 消费者
    go func() {
        for msg := range ch {  // channel 关闭后自动退出
            process(msg)
        }
        close(done)
    }()
    
    <-done
}
```

**实际案例**：日志异步写入

```go
type AsyncLogger struct {
    entries chan LogEntry
    done    chan struct{}
}

func NewAsyncLogger() *AsyncLogger {
    l := &AsyncLogger{
        entries: make(chan LogEntry, 1000),
        done:    make(chan struct{}),
    }
    
    // 后台协程持续消费日志
    go func() {
        for entry := range l.entries {
            writeFile(entry)
        }
        close(l.done)
    }()
    
    return l
}

func (l *AsyncLogger) Log(msg string) {
    l.entries <- LogEntry{Msg: msg, Time: time.Now()}
}

func (l *AsyncLogger) Close() {
    close(l.entries)  // 停止接收新日志
    <-l.done          // 等待所有日志写入完成
}
```

### 2.3 定时任务模式

**本质**：通过 `time.Ticker` 实现周期性任务

```go
// ✅ 定时任务 + 优雅退出
func ScheduledTask() {
    done := make(chan struct{})
    ticker := time.NewTicker(5 * time.Second)
    
    go func() {
        defer ticker.Stop()
        
        for {
            select {
            case <-done:
                fmt.Println("定时任务停止")
                return
            case <-ticker.C:
                doPeriodicWork()
            }
        }
    }()
    
    // 10秒后停止
    time.Sleep(10 * time.Second)
    close(done)
}
```

**实际案例**：定期清理空闲连接

```go
type ConnectionManager struct {
    conns    map[string]*Connection
    mu       sync.Mutex
    done     chan struct{}
}

func (m *ConnectionManager) StartCleanup() {
    ticker := time.NewTicker(30 * time.Second)
    
    go func() {
        defer ticker.Stop()
        
        for {
            select {
            case <-m.done:
                return
            case <-ticker.C:
                m.cleanupIdleConnections()
            }
        }
    }()
}

func (m *ConnectionManager) cleanupIdleConnections() {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    now := time.Now()
    for id, conn := range m.conns {
        if now.Sub(conn.LastActive) > 5*time.Minute {
            conn.Close()
            delete(m.conns, id)
        }
    }
}
```

### 2.4 重试模式

**本质**：通过循环 + 退避策略实现容错

```go
// ✅ 指数退避重试
func RetryWithBackoff(maxRetries int, operation func() error) error {
    var err error
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        err = operation()
        if err == nil {
            return nil
        }
        
        // 指数退避：1s, 2s, 4s, 8s...
        backoff := time.Duration(1<<uint(attempt)) * time.Second
        time.Sleep(backoff)
    }
    
    return fmt.Errorf("重试%d次后失败: %w", maxRetries, err)
}

// 使用示例
err := RetryWithBackoff(5, func() error {
    return httpClient.Get("https://api.example.com/data")
})
```

### 2.5 状态机模式

**本质**：通过 for + select 实现复杂状态流转

```go
// ✅ 连接状态机
type ConnectionState int

const (
    StateDisconnected ConnectionState = iota
    StateConnecting
    StateConnected
    StateReconnecting
)

func ConnectionStateMachine() {
    state := StateDisconnected
    events := make(chan Event)
    done := make(chan struct{})
    
    go func() {
        for {
            select {
            case <-done:
                return
            
            case event := <-events:
                switch state {
                case StateDisconnected:
                    if event.Type == EventConnect {
                        state = StateConnecting
                        go attemptConnect()
                    }
                
                case StateConnecting:
                    if event.Type == EventConnectSuccess {
                        state = StateConnected
                    } else if event.Type == EventConnectFail {
                        state = StateReconnecting
                    }
                
                case StateConnected:
                    if event.Type == EventDisconnect {
                        state = StateReconnecting
                    }
                
                case StateReconnecting:
                    if event.Type == EventReconnectSuccess {
                        state = StateConnected
                    }
                }
            }
        }
    }()
}
```

### 2.6 心跳检测模式

**本质**：通过定时发送心跳 + 超时检测判断连接活性

```go
// ✅ 心跳检测
func HeartbeatCheck(conn *Connection, timeout time.Duration) {
    heartbeat := time.NewTicker(5 * time.Second)
    done := make(chan struct{})
    lastPong := time.Now()
    
    go func() {
        defer heartbeat.Stop()
        
        for {
            select {
            case <-done:
                return
            
            case <-heartbeat.C:
                // 检查是否超时
                if time.Since(lastPong) > timeout {
                    conn.Close()
                    return
                }
                
                // 发送心跳
                conn.Send(PingMessage)
            
            case <-conn.PongChannel:
                lastPong = time.Now()  // 更新最后心跳时间
            }
        }
    }()
}
```

---

## 三、范式2：sync.WaitGroup 的 5 种核心编码方式

### 3.1 批量并行任务

**本质**：将串行任务并行化，加速执行

```go
// ✅ 并行处理多个独立任务
func ParallelTasks() {
    urls := []string{
        "https://api1.example.com",
        "https://api2.example.com",
        "https://api3.example.com",
    }
    
    var wg sync.WaitGroup
    results := make([]string, len(urls))
    
    for i, url := range urls {
        wg.Add(1)
        go func(idx int, u string) {
            defer wg.Done()
            
            resp, err := http.Get(u)
            if err != nil {
                results[idx] = fmt.Sprintf("error: %v", err)
                return
            }
            results[idx] = resp.Status
        }(i, url)  // 注意：必须传参，避免闭包陷阱
    }
    
    wg.Wait()  // 等待所有请求完成
    fmt.Println(results)
}
```

### 3.2 Fan-out / Fan-in 模式

**本质**：分发任务（Fan-out）+ 收集结果（Fan-in）

```go
// ✅ Fan-out/Fan-in 完整示例
func FanOutFanIn() {
    // 输入数据
    inputs := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
    
    // Fan-out：启动多个 worker
    var wg sync.WaitGroup
    resultCh := make(chan int, len(inputs))
    
    for _, input := range inputs {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            // 模拟耗时计算
            time.Sleep(100 * time.Millisecond)
            resultCh <- n * n
        }(input)
    }
    
    // 等待所有 worker 完成，关闭结果 channel
    go func() {
        wg.Wait()
        close(resultCh)
    }()
    
    // Fan-in：收集所有结果
    var results []int
    for result := range resultCh {
        results = append(results, result)
    }
    
    fmt.Printf("结果: %v\n", results)
}
```

### 3.3 Worker Pool 模式

**本质**：限制并发数，避免资源耗尽

```go
// ✅ Worker Pool：固定数量的 worker 处理任务队列
func WorkerPool(workerCount int, tasks []func()) {
    taskCh := make(chan func(), len(tasks))
    var wg sync.WaitGroup
    
    // 启动固定数量的 worker
    for i := 0; i < workerCount; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            
            for task := range taskCh {
                fmt.Printf("Worker %d 执行任务\n", workerID)
                task()
            }
        }(i)
    }
    
    // 发送所有任务
    for _, task := range tasks {
        taskCh <- task
    }
    close(taskCh)  // 任务发送完毕，关闭 channel
    
    wg.Wait()  // 等待所有 worker 完成
}

// 使用示例
tasks := make([]func(), 100)
for i := 0; i < 100; i++ {
    n := i
    tasks[i] = func() {
        time.Sleep(50 * time.Millisecond)
        fmt.Printf("任务 %d 完成\n", n)
    }
}

// 只用 5 个 worker 并发执行
WorkerPool(5, tasks)
```

### 3.4 错误收集模式（errgroup）

**本质**：WaitGroup + 错误处理的组合

```go
// ✅ 使用 errgroup（官方扩展包）
import "golang.org/x/sync/errgroup"

func ErrGroupExample() error {
    g, ctx := errgroup.WithContext(context.Background())
    
    urls := []string{
        "https://api1.example.com",
        "https://api2.example.com",
        "https://api3.example.com",
    }
    
    for _, url := range urls {
        url := url  // 闭包捕获
        g.Go(func() error {
            req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
            resp, err := http.DefaultClient.Do(req)
            if err != nil {
                return err
            }
            defer resp.Body.Close()
            
            if resp.StatusCode != 200 {
                return fmt.Errorf("%s 返回 %d", url, resp.StatusCode)
            }
            return nil
        })
    }
    
    // 等待所有任务完成，返回第一个错误
    if err := g.Wait(); err != nil {
        return err
    }
    
    fmt.Println("所有请求成功")
    return nil
}
```

### 3.5 分阶段执行模式

**本质**：多个阶段，每个阶段内部并行，阶段间串行

```go
// ✅ 分阶段处理：数据获取 → 数据处理 → 数据保存
func MultiStagePipeline() {
    // 阶段1：并行获取数据
    var wg1 sync.WaitGroup
    dataCh := make(chan Data, 10)
    
    sources := []string{"db1", "db2", "db3"}
    for _, source := range sources {
        wg1.Add(1)
        go func(src string) {
            defer wg1.Done()
            data := fetchData(src)
            dataCh <- data
        }(source)
    }
    
    wg1.Wait()
    close(dataCh)
    
    // 阶段2：并行处理数据
    var wg2 sync.WaitGroup
    resultCh := make(chan Result, 10)
    
    for data := range dataCh {
        wg2.Add(1)
        go func(d Data) {
            defer wg2.Done()
            result := processData(d)
            resultCh <- result
        }(data)
    }
    
    wg2.Wait()
    close(resultCh)
    
    // 阶段3：保存结果
    for result := range resultCh {
        saveResult(result)
    }
}
```

---

## 四、两种范式的组合模式

### 4.1 长驻服务 + 批量任务

**本质**：`go + for{}` 作为服务主循环，内部用 `WaitGroup` 处理批量请求

```go
// ✅ HTTP 服务器模式
type Server struct {
    handler Handler
    done    chan struct{}
}

func (s *Server) Start() {
    listener, _ := net.Listen("tcp", ":8080")
    
    // 长驻协程：持续接受连接
    go func() {
        for {
            select {
            case <-s.done:
                listener.Close()
                return
            default:
                conn, err := listener.Accept()
                if err != nil {
                    continue
                }
                
                // 为每个连接启动处理协程（短任务）
                go s.handleConnection(conn)
            }
        }
    }()
}

func (s *Server) handleConnection(conn net.Conn) {
    defer conn.Close()
    
    // 使用 WaitGroup 等待该连接的所有子任务
    var wg sync.WaitGroup
    
    for {
        request := readRequest(conn)
        if request == nil {
            break
        }
        
        wg.Add(1)
        go func(req Request) {
            defer wg.Done()
            response := s.handler.Handle(req)
            writeResponse(conn, response)
        }(request)
    }
    
    wg.Wait()  // 等待该连接的所有请求处理完成
}
```

### 4.2 消息队列消费者

**本质**：`go + for{}` 持续消费消息，`WaitGroup` 确保优雅关闭

```go
// ✅ Kafka 消费者模式
type Consumer struct {
    consumer *kafka.Consumer
    handler  MessageHandler
    wg       sync.WaitGroup
    done     chan struct{}
}

func (c *Consumer) Start(workerCount int) {
    // 长驻协程：持续从 Kafka 拉取消息
    go func() {
        for {
            select {
            case <-c.done:
                return
            default:
                msg := c.consumer.ReadMessage()
                
                // 投递给 worker 处理
                c.wg.Add(1)
                go func(m *kafka.Message) {
                    defer c.wg.Done()
                    c.handler.Handle(m)
                }(msg)
            }
        }
    }()
}

func (c *Consumer) Stop() {
    close(c.done)        // 停止接收新消息
    c.wg.Wait()          // 等待所有正在处理的消息完成
    c.consumer.Close()
}
```

### 4.3 定时批处理

**本质**：`go + for{}` + `time.Ticker` 定时触发，`WaitGroup` 批量处理

```go
// ✅ 定时批处理：每秒批量处理一次
type BatchProcessor struct {
    buffer   chan Task
    ticker   *time.Ticker
    done     chan struct{}
}

func (bp *BatchProcessor) Start() {
    bp.ticker = time.NewTicker(time.Second)
    
    // 长驻协程：定时触发批处理
    go func() {
        defer bp.ticker.Stop()
        
        for {
            select {
            case <-bp.done:
                // 处理剩余任务
                bp.processBatch()
                return
            
            case task := <-bp.buffer:
                // 收集任务（非阻塞）
                select {
                case bp.currentBatch = append(bp.currentBatch, task):
                default:
                    // buffer 满了，立即处理
                    bp.processBatch()
                    bp.currentBatch = append(bp.currentBatch, task)
                }
            
            case <-bp.ticker.C:
                // 定时触发批处理
                bp.processBatch()
            }
        }
    }()
}

func (bp *BatchProcessor) processBatch() {
    if len(bp.currentBatch) == 0 {
        return
    }
    
    var wg sync.WaitGroup
    for _, task := range bp.currentBatch {
        wg.Add(1)
        go func(t Task) {
            defer wg.Done()
            processTask(t)
        }(task)
    }
    
    wg.Wait()  // 等待当前批次处理完成
    bp.currentBatch = nil
}
```

---

## 五、核心关键点总结

### 5.1 万能公式

```
高并发编程 = Goroutine（执行） + Channel（通信） + 同步原语（协调）
```

### 5.2 选择指南

```
需要长期运行的服务/监听器？
  ├─ 用 go func() + for{} + select
  │   ├─ 事件驱动 → 监听多个 channel
  │   ├─ 定时任务 → 加 time.Ticker
  │   └─ 优雅退出 → close(done channel)
  │
需要批量处理/并行加速？
  ├─ 用 sync.WaitGroup
  │   ├─ 简单并行 → wg.Add + wg.Done + wg.Wait
  │   ├─ 收集结果 → 加 channel
  │   ├─ 错误处理 → 用 errgroup
  │   └─ 限制并发 → Worker Pool 模式
  │
两者都需要？
  └─ 组合使用：长驻服务接收请求，WaitGroup 处理批量任务
```

### 5.3 避坑清单

| 陷阱 | 问题 | 正确做法 |
|------|------|---------|
| **协程泄漏** | `go func() { for {} }()` 没有退出机制 | 必须用 done channel 或 context 控制退出 |
| **闭包陷阱** | `go func() { fmt.Println(i) }()` | 传参：`go func(i int) { ... }(i)` |
| **WaitGroup 误用** | 在 goroutine 内 wg.Add(1) | 必须在 goroutine 外 Add |
| **重复 close** | 多次 close(channel) 导致 panic | 用 select+default 或 sync.Once |
| **channel 死锁** | 无缓冲 channel 发送后没有接收方 | 确保有对应的接收方，或用缓冲 |
| **资源泄漏** | 忘记 defer 清理资源 | 始终用 defer 关闭文件/连接/channel |

### 5.4 终极心法

```
1. 先想清楚：是长驻服务还是批量任务？
2. 长驻用 for + select，批量用 WaitGroup
3. 所有协程必须有退出机制（done / context）
4. 共享数据必须加锁或用 channel 通信
5. 用 go test -race 检测数据竞争
6. 用 pprof 检测协程泄漏
```

---

## 六、实战练习：从简单到复杂

### 练习1：并行下载文件（WaitGroup）

```go
func DownloadFiles(urls []string) error {
    var wg sync.WaitGroup
    errCh := make(chan error, len(urls))
    
    for _, url := range urls {
        wg.Add(1)
        go func(u string) {
            defer wg.Done()
            resp, err := http.Get(u)
            if err != nil {
                errCh <- err
                return
            }
            defer resp.Body.Close()
            saveToFile(resp.Body, u)
        }(url)
    }
    
    wg.Wait()
    close(errCh)
    
    // 检查是否有错误
    for err := range errCh {
        if err != nil {
            return err
        }
    }
    return nil
}
```

### 练习2：实时日志分析器（go + for{}）

```go
type LogAnalyzer struct {
    logs    chan string
    done    chan struct{}
    stats   map[string]int
    mu      sync.Mutex
}

func (la *LogAnalyzer) Start() {
    go func() {
        for {
            select {
            case <-la.done:
                la.printStats()
                return
            case log := <-la.logs:
                la.mu.Lock()
                la.stats[extractKeyword(log)]++
                la.mu.Unlock()
            }
        }
    }()
}

func (la *LogAnalyzer) Analyze(log string) {
    la.logs <- log
}

func (la *LogAnalyzer) Stop() {
    close(la.done)
}
```

### 练习3：并发爬虫（组合模式）

```go
func Crawl(startURL string, maxDepth int) {
    type Task struct {
        URL   string
        Depth int
    }
    
    var wg sync.WaitGroup
    taskCh := make(chan Task, 100)
    visited := sync.Map{}
    
    // 启动 worker pool
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for task := range taskCh {
                if _, loaded := visited.LoadOrStore(task.URL, true); loaded {
                    continue
                }
                
                links := fetchLinks(task.URL)
                if task.Depth < maxDepth {
                    for _, link := range links {
                        taskCh <- Task{URL: link, Depth: task.Depth + 1}
                    }
                }
            }
        }()
    }
    
    // 发送初始任务
    taskCh <- Task{URL: startURL, Depth: 0}
    
    // 等待所有任务完成
    wg.Wait()
    close(taskCh)
}
```

---

## 七、进阶：并发模式的演进

### 7.1 从裸奔到优雅

```go
// ❌ Level 0：裸奔（不推荐）
go func() {
    for {
        // 无退出机制，协程泄漏
    }
}()

// ⚠️ Level 1：基础退出
done := make(chan struct{})
go func() {
    for {
        select {
        case <-done:
            return
        }
    }
}()
close(done)

// ✅ Level 2：Context 控制
ctx, cancel := context.WithCancel(context.Background())
go func() {
    for {
        select {
        case <-ctx.Done():
            return
        }
    }
}()
cancel()

// 🌟 Level 3：errgroup 优雅并发
g, ctx := errgroup.WithContext(context.Background())
g.Go(func() error {
    return worker1(ctx)
})
g.Go(func() error {
    return worker2(ctx)
})
if err := g.Wait(); err != nil {
    log.Fatal(err)
}
```

### 7.2 从同步到异步

```go
// ❌ 同步阻塞
result := slowOperation()  // 阻塞主流程

// ✅ 异步 + channel
ch := make(chan Result)
go func() {
    ch <- slowOperation()
}()
result := <-ch  // 需要时再取

// 🌟 异步 + Future 模式
func AsyncOperation() <-chan Result {
    ch := make(chan Result, 1)
    go func() {
        ch <- slowOperation()
    }()
    return ch
}

// 使用
future := AsyncOperation()
// ... 做其他事情
result := <-future
```

---

## 八、性能优化要点

### 8.1 Goroutine 数量控制

```go
// ❌ 无限制创建
for _, task := range tasks {
    go process(task)  // 可能创建百万协程
}

// ✅ 限制并发数
sem := make(chan struct{}, 100)  // 信号量
for _, task := range tasks {
    sem <- struct{}{}  // 获取许可
    go func(t Task) {
        defer func() { <-sem }()  // 释放许可
        process(t)
    }(task)
}
```

### 8.2 Channel 缓冲大小

```
原则：缓冲大小 = 生产者和消费者的速率差 × 处理时间

缓冲=0：严格同步，零堆积
缓冲=1：最小缓冲，避免死锁（推荐默认）
缓冲=N：允许短暂堆积，但要注意内存
```

### 8.3 避免 Goroutine 泄漏

```go
// ✅ 始终确保 goroutine 能退出
func SafeGoroutine() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    go func() {
        for {
            select {
            case <-ctx.Done():
                return  // 超时自动退出
            default:
                doWork()
            }
        }
    }()
}
```

---

## 九、一句话总结

```
go func() + for{} = 长驻服务（持续监听/处理）
sync.WaitGroup     = 批量任务（并行加速/等待完成）
两者组合           = 完整的高并发架构

底层核心 = Goroutine（执行） + Channel（通信） + 同步原语（协调）
退出机制 = done channel / context（必须有！）
并发安全 = 加锁 / atomic / channel（三选一）
```

**掌握这些，你就掌握了 Go 并发编程的 99%！**
