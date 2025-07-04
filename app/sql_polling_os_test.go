package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOSVersion(t *testing.T) {
	assert := assert.New(t)
	type test struct {
		got  string
		os   string
		arch string
	}
	tests := []test{
		{got: `Microsoft SQL Server 2014 (SP2-GDR) (KB4505217) - 12.0.5223.6 (X64)
		May 26 2019 20:36:50
		Copyright (c) Microsoft Corporation
		Developer Edition (64-bit) on Windows NT 6.3 <X64> (Build 19045: ) (Hypervisor)`,
			os: "Windows NT 6.3", arch: "x64"},

		{got: `Microsoft SQL Server 2014 (SP2-GDR) (KB4505217) - 12.0.5223.6 (X64)
		May 26 2019 20:36:50
		Copyright (c) Microsoft Corporation
		Developer Edition (64-bit) onx Windows NT 6.3 <X64> (Build 19045: ) (Hypervisor)`,
			os: "unknown", arch: "unknown"},

		{got: `Microsoft SQL Server 2016 (SP2-GDR) (KB4583460) - 13.0.5108.50 (X64) 
			May 20 2022 20:28:29 
			Copyright (c) Microsoft Corporation
			Developer Edition (64-bit) on Windows 10 Pro 10.0 <X64> (Build 19045: ) (Hypervisor)`,
			os: "Windows 10 Pro 10.0", arch: "x64"},

		{got: `Microsoft SQL Server 2016 (SP3) (KB5003279) - 13.0.6300.2 (X64) 
		Aug  7 2021 01:20:37 
		Copyright (c) Microsoft Corporation
		Enterprise Edition: Core-based Licensing (64-bit) on Windows Server 2012 R2 Standard 6.3 <X64> (Build 9600: )`,
			os: "Windows Server 2012 R2 Standard 6.3", arch: "x64"},

		{got: `Microsoft SQL Server 2016 (SP3) (KB5003279) - 13.0.6300.2 (X64) 
		Aug  7 2021 01:20:37 
		Copyright (c) Microsoft Corporation
		Enterprise Edition: Core-based Licensing (64-bit) on Windows Server 2019 Standard 10.0 <X64> (Build 17763: ) (Hypervisor)`,
			os: "Windows Server 2019 Standard 10.0", arch: "x64"},

		{got: `Microsoft SQL Server 2016 (SP3) (KB5003279) - 13.0.6300.2 (X64) 
		Aug  7 2021 01:20:37 
		Copyright (c) Microsoft Corporation
		Enterprise Edition: Core-based Licensing (64-bit) on Windows Server 2012 R2 Standard 6.3 <X64> (Build 9600: ) (Hypervisor)`,
			os: "Windows Server 2012 R2 Standard 6.3", arch: "x64"},

		{got: `Microsoft SQL Server 2019 (RTM-CU21) (KB5025808) - 15.0.4316.3 (X64) 
		Jun  1 2023 16:32:31 
		Copyright (C) 2019 Microsoft Corporation
		Standard Edition (64-bit) on Windows Server 2019 Standard 10.0 <X64> (Build 17763: ) (Hypervisor)`,
			os: "Windows Server 2019 Standard 10.0", arch: "x64"},

		{got: `Microsoft SQL Server 2022 (RTM-CU7) (KB5028743) - 16.0.4065.3 (X64) 
		Jul 25 2023 18:03:43 
		Copyright (C) 2022 Microsoft Corporation
		Express Edition (64-bit) on Linux (Ubuntu 20.04.6 LTS) <X64>`,
			os: "Linux (Ubuntu 20.04.6 LTS)", arch: "x64"},

		{got: ``,
			os: "unknown", arch: "unknown"},
	}
	for _, tc := range tests {
		v := parseatatversion(tc.got)
		assert.Equal(tc.os, v.os)
		assert.Equal(tc.arch, v.arch)
	}
}
