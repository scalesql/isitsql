package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHeadBlocker(t *testing.T) {
	assert := assert.New(t)
	m := map[int16]int16{
		11: 10,
		12: 11,
	}

	hb, _, _, _ := headBlocker(12, m, []int16{}, 0, "")
	assert.Equal(int16(10), hb)
	hb, _, _, _ = headBlocker(11, m, []int16{}, 0, "")
	assert.Equal(int16(10), hb)
	hb, _, _, _ = headBlocker(10, m, []int16{}, 0, "")
	assert.Equal(int16(10), hb)
	hb, _, _, _ = headBlocker(9, m, []int16{}, 0., "")
	assert.Equal(int16(9), hb)
}

func TestHeadBlockerDepthBad(t *testing.T) {
	assert := assert.New(t)
	m := make(map[int16]int16)
	for i := int16(10); i <= 200; i++ {
		m[i] = int16(i - 1)
	}
	// start at the end and go up
	total, _, _, err := headBlocker(200, m, []int16{}, 0, "")
	assert.Error(err)
	assert.Equal(int16(0), total)
}

func TestHeadBlockerDeadlock(t *testing.T) {
	assert := assert.New(t)
	m := map[int16]int16{
		11: 10,
		12: 11,
		10: 12,
	}

	hb, _, _, err := headBlocker(6, m, []int16{}, 0, "")
	assert.NoError(err)
	assert.Equal(int16(6), hb)

	hb, _, _, err = headBlocker(12, m, []int16{}, 0, "")
	assert.NoError(err)
	assert.Equal(int16(12), hb)
}

func TestSmallBlocking(t *testing.T) {
	assert := assert.New(t)
	fs := []Session{
		{SessionID: 1, BlockerID: 0},
		{SessionID: 2, BlockerID: 1},
		{SessionID: 3, BlockerID: 2},
		{SessionID: 4, BlockerID: 2},
	}
	err := populateBlocking(fs)
	assert.NoError(err)
	assert.Equal(3, fs[0].TotalBlocked)
	assert.Equal(2, fs[1].TotalBlocked)
}

func TestTotalBlocked(t *testing.T) {
	assert := assert.New(t)
	fs := []Session{
		{SessionID: 1, BlockerID: 0},
		{SessionID: 2, BlockerID: 1},
		{SessionID: 3, BlockerID: 2},
		{SessionID: 4, BlockerID: 2},
		{SessionID: 5, BlockerID: 2},
		{SessionID: 6, BlockerID: 5},
		{SessionID: 7, BlockerID: 1},
		{SessionID: 8, BlockerID: 6},
	}
	err := populateBlocking(fs)
	assert.NoError(err)
	assert.Equal(7, fs[0].TotalBlocked)
	assert.Equal(5, fs[1].TotalBlocked)
	assert.Equal(0, fs[2].TotalBlocked)
	assert.Equal(0, fs[3].TotalBlocked)
	assert.Equal(2, fs[4].TotalBlocked)
	assert.Equal(1, fs[5].TotalBlocked)
	assert.Equal(0, fs[6].TotalBlocked)
}

// https://justin.azoff.dev/blog/ensuring-zero-allocations-in-go-tests/
func TestAllocations(t *testing.T) {
	res := testing.Benchmark(BenchmarkMemory)
	allocs := res.AllocedBytesPerOp()
	assert.LessOrEqual(t, allocs, int64(3000))
}

func BenchmarkBigMemory(b *testing.B) {
	// BenchmarkBigMemory-8   	      21	  52270333 ns/op	  188462 B/op	   19237 allocs/op
	// 200:  BenchmarkBigMemory-8   	   14134	     84208 ns/op	   10821 B/op	    1129 allocs/op
	// 500:  BenchmarkBigMemory-8   	    4028	    266378 ns/op	   37698 B/op	    3488 allocs/op
	// 750:  BenchmarkBigMemory-8   	    2906	    392362 ns/op	   52366 B/op	    5498 allocs/op
	// 1000: BenchmarkBigMemory-8   	    2102	    541603 ns/op	   80754 B/op	    7509 allocs/op
	// 2000: BenchmarkBigMemory-8   	    1078	   1080667 ns/op	  175377 B/op	   15533 allocs/op
	// 5000: BenchmarkBigMemory-8   	     421	   2836771 ns/op	  441258 B/op	   39567 allocs/op
	fs := []Session{
		{SessionID: 1, BlockerID: 0},
		{SessionID: 2, BlockerID: 1},
		{SessionID: 3, BlockerID: 2},
		{SessionID: 4, BlockerID: 2},
		{SessionID: 5, BlockerID: 2},
		{SessionID: 6, BlockerID: 5},
		{SessionID: 7, BlockerID: 1},
		{SessionID: 8, BlockerID: 6},
	}
	for i := int16(50); i <= int16(5000); i++ {
		fs = append(fs, Session{SessionID: i, BlockerID: 2})
	}

	for n := 0; n < b.N; n++ {
		err := populateBlocking(fs)
		assert.NoError(b, err)
	}
}

// count=100			BenchmarkDeepTree-8   	    1041	   1140454 ns/op	   615379 B/op	   10606 allocs/op
// count=500 (80MB)		BenchmarkDeepTree-8   	      22	  49767959 ns/op	 88198176 B/op	  284346 allocs/op
// count=1000 (715MB)	BenchmarkDeepTree-8   	       4	 289308925 ns/op	715117990 B/op	 1288250 allocs/op
func BenchmarkDeepTree(b *testing.B) {
	count := 100
	fs := make([]Session, 0, count)
	for i := int16(1); i <= int16(count); i++ {
		fs = append(fs, Session{SessionID: i, BlockerID: i - 1})
	}
	for n := 0; n < b.N; n++ {
		err := populateBlocking(fs)
		assert.NoError(b, err)
	}
	assert.Equal(b, count-1, fs[0].TotalBlocked)
}

// Baseline: 	BenchmarkMemory-8   	       1	3781581100 ns/op 15258161408 B/op	    2391 allocs/op (15.25 GB)
// NoOp:		BenchmarkMemory-8   	508316696	     2.345 ns/op	       0 B/op	       0 allocs/op
// Rewrite:     BenchmarkMemory-8   	   41192	     31643 ns/op	    2757 B/op	     247 allocs/op
func BenchmarkMemory(b *testing.B) {

	b.ReportAllocs()
	fakeSessions := []Session{
		{SessionID: 59, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 57, 337000000, time.UTC), RunTimeSeconds: 1185, RunTimeText: "19m", Status: "suspended", StatementText: "WAITFOR DELAY '00:59:00'\n\t\t", Database: "master", WaitType: "WAITFOR", WaitTime: 1185174, WaitResource: "", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "WAITFOR", OpenTxnCount: 1, BlockerID: 0, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 60, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 340000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1184177, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 59, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 61, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 353000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1184162, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},

		{SessionID: 71, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 510000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1184008, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 76, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 587000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183929, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 86, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 760000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183758, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 78, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 633000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183882, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 69, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 480000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1184036, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 80, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 667000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183852, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 88, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 790000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183727, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 75, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 573000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183944, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 67, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 450000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1184067, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 73, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 540000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183975, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 66, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 433000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1184082, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 79, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 650000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183867, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 63, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 387000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1184129, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 62, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 370000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1184145, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 81, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 680000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183836, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 64, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 403000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1184113, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 89, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 807000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183711, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 70, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 503000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1184012, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 82, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 697000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183821, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 83, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 717000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183802, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 68, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 467000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1184051, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 65, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 420000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1184098, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 87, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 777000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183741, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 72, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 527000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183990, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 84, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 730000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183789, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 85, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 743000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183774, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 74, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 557000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183959, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
		{SessionID: 77, RequestID: 0, HasRequest: true, StartTime: time.Date(2024, time.April, 21, 10, 25, 58, 603000000, time.UTC), RunTimeSeconds: 1184, RunTimeText: "19m", Status: "suspended", StatementText: "SELECT TOP 10 *     FROM\tAdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK)", Database: "master", WaitType: "LCK_M_SIX", WaitTime: 1183913, WaitResource: "OBJECT: 6:2085582468:0 ", HostName: "D40", AppName: "makeblock.exe", LoginName: "MicrosoftAccount\\graz@sqlteam.com", PercentComplete: 0, Command: "SELECT", OpenTxnCount: 1, BlockerID: 60, HeadBlockerID: 0, TotalBlocked: 0, Depth: 0},
	}
	for n := 0; n < b.N; n++ {
		err := populateBlocking(fakeSessions)
		assert.NoError(b, err)
	}
}
