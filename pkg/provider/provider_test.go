package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	mtype1CPU1Gi = &MachineType{
		Name:           "mtype1CPU1Gi",
		Memory:         "1Gi",
		CPU:            "1",
		MemoryResource: resource.MustParse("1Gi"),
		CPUResource:    resource.MustParse("1"),
	}
	mtype2CPU8Gi = &MachineType{
		Name:           "mtype2CPU8Gi",
		Memory:         "8Gi",
		CPU:            "2",
		MemoryResource: resource.MustParse("8Gi"),
		CPUResource:    resource.MustParse("2"),
	}
	mtype4CPU2Gi = &MachineType{
		Name:           "mtype4CPU2Gi",
		Memory:         "2Gi",
		CPU:            "4",
		MemoryResource: resource.MustParse("2Gi"),
		CPUResource:    resource.MustParse("4"),
	}
	mtype4CPU4Gi = &MachineType{
		Name:           "mtype4CPU4Gi",
		Memory:         "4Gi",
		CPU:            "4",
		MemoryResource: resource.MustParse("4Gi"),
		CPUResource:    resource.MustParse("4"),
	}
	mtype4CPU8Gi = &MachineType{
		Name:           "mtype4CPU8Gi",
		Memory:         "8Gi",
		CPU:            "4",
		MemoryResource: resource.MustParse("8Gi"),
		CPUResource:    resource.MustParse("4"),
	}
	mtype1CPU1GiPrice1 = &MachineType{
		Name:           "mtype1CPU1Gi",
		Memory:         "1Gi",
		CPU:            "1",
		MemoryResource: resource.MustParse("1Gi"),
		CPUResource:    resource.MustParse("1"),
		PriceHour:      1,
	}
)

func TestSortedMachineTypes(t *testing.T) {
	tcs := []struct {
		in       []*MachineType
		expected []*MachineType
	}{
		{
			in:       []*MachineType{mtype4CPU8Gi, mtype4CPU4Gi, mtype2CPU8Gi, mtype4CPU2Gi, mtype1CPU1Gi},
			expected: []*MachineType{mtype4CPU8Gi, mtype4CPU4Gi, mtype4CPU2Gi, mtype2CPU8Gi, mtype1CPU1Gi},
		},
		{
			in:       []*MachineType{mtype1CPU1Gi, mtype2CPU8Gi, mtype4CPU2Gi, mtype4CPU4Gi, mtype4CPU8Gi},
			expected: []*MachineType{mtype4CPU8Gi, mtype4CPU4Gi, mtype4CPU2Gi, mtype2CPU8Gi, mtype1CPU1Gi},
		},
		{
			in:       []*MachineType{mtype1CPU1Gi, mtype1CPU1GiPrice1, mtype2CPU8Gi, mtype4CPU2Gi, mtype4CPU4Gi, mtype4CPU8Gi},
			expected: []*MachineType{mtype4CPU8Gi, mtype4CPU4Gi, mtype4CPU2Gi, mtype2CPU8Gi, mtype1CPU1Gi, mtype1CPU1GiPrice1},
		},
	}

	for i, tc := range tcs {
		require.Equalf(t, tc.expected, SortedMachineTypes(tc.in), "TC#%d", i+1)
	}
}
