package persist

type GlobalSync[T GlobalModel] struct {
	Data   T
	Op     int8
	BitSet GlobalBitSet[T]
}
