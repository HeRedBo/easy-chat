package bitmap

type Bitmap struct {
	bits []byte
	size int
}

func NewBitmap(size int) *Bitmap {
	if size == 0 {
		size = 250
	}
	return &Bitmap{
		bits: make([]byte, size),
		size: size * 8,
	}
}

func (b *Bitmap) Set(id string) {
	// id 在那个bit 上
	idx := hash(id) % b.size
	//计算在那额byte
	byteIdx := idx / 8
	// 在这个byte中的那个bit位置
	bitIdx := idx % 8
	b.bits[byteIdx] |= 1 << bitIdx
}

func (b *Bitmap) IsSet(id string) bool {
	idx := hash(id) % b.size
	// 计算在那个byte
	byteIdx := idx / 8
	// 在这个byte中的那个bit位置
	bitIdx := idx % 8
	return (b.bits[byteIdx] & (1 << bitIdx)) != 0
}

func (b *Bitmap) Export() []byte {
	return b.bits
}

func Load(bits []byte) *Bitmap {
	if len(bits) == 0 {
		return NewBitmap(0)
	}

	return &Bitmap{
		bits: bits,
		size: len(bits) * 8,
	}
}

func hash(id string) int {
	// 使用BKDR哈希算法
	seed := 131313 // 31 131 1313 13131 131313, etc
	hash := 0
	for _, c := range id {
		hash = hash*seed + int(c)
	}
	return hash & 0x7FFFFFFF
}

// Count 统计所有为1的位的数量（在线人数/打卡人数）
func (b *Bitmap) Count() uint64 {
	var count uint64
	// 遍历每个字节，统计其中1的个数
	for _, b := range b.bits {
		// 快速统计一个字节中1的个数（Go内置技巧）
		count += uint64(popCount(b))
	}
	return count
}

// popCount 统计单个字节中1的位数（辅助函数）
func popCount(b byte) int {
	count := 0
	for b != 0 {
		count++
		b &= b - 1 // 清除最低位的1
	}
	return count
}
