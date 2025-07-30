package tdxsetup

import (
	"context"
	"tdx-init/pkg/disk"
)

type DiskInitializerer interface {
	FindDisk(ctx context.Context) (string, error)
}

type LargestDiskInitializer struct{}

func (l *LargestDiskInitializer) FindDisk(_ context.Context) (string, error) {
	return disk.FindLargestDisk()
}

type PathGlobDiskInitializer struct {
	PathGlob string
}

func (p *PathGlobDiskInitializer) FindDisk(_ context.Context) (string, error) {
	return disk.FindFirstDiskByPathGlob(p.PathGlob)
}
