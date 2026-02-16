package testutil

// LogDataset defines a named set of log lines for pattern-detection tests.
type LogDataset struct {
	Name        string   // test name
	Lines       []string // raw log lines
	WantNew     int      // expected count of "new" pattern first-seen
	WantKnown   int      // expected count of "known" (same pattern seen again)
	WantPatterns int    // expected number of distinct patterns in store (min)
}

// Datasets returns multiple simulated log datasets for testing pattern detection.
func Datasets() []LogDataset {
	return []LogDataset{
		{
			Name:        "MixedLevels",
			Lines:       SampleLogLines(),
			WantNew:     3,
			WantKnown:   1,
			WantPatterns: 3,
		},
		{
			Name: "SamePatternRepeated",
			Lines: []string{
				"ERROR connection refused from 10.0.0.1",
				"ERROR connection refused from 10.0.0.2",
				"ERROR connection refused from 10.0.0.3",
			},
			WantNew:     1,
			WantKnown:   2,
			WantPatterns: 1,
		},
		{
			Name: "AllDistinct",
			Lines: []string{
				"ERROR disk full",
				"WARN timeout",
				"INFO started",
				"DEBUG trace entry",
			},
			WantNew:     4,
			WantKnown:   0,
			WantPatterns: 4,
		},
		{
			Name: "OnlyErrors",
			Lines: []string{
				"ERROR failed to open file 123",
				"ERROR failed to open file 456",
				"ERROR out of memory",
			},
			WantNew:     2,
			WantKnown:   1,
			WantPatterns: 2,
		},
		{
			Name: "WithEmptyAndWhitespace",
			Lines: []string{
				"ERROR first",
				"",
				"ERROR first",
				"   ",
				"WARN second",
			},
			WantNew:     2,
			WantKnown:   1,
			WantPatterns: 2,
		},
		{
			Name: "LevelInMiddle",
			Lines: []string{
				"request failed with error code 500",
				"warning: retry attempt 1",
			},
			WantNew:     2,
			WantKnown:   0,
			WantPatterns: 2,
		},
		{
			Name: "UUIDAndHexDropped",
			Lines: []string{
				"ERROR transaction a1b2c3d4-e5f6-7890-abcd-ef1234567890 failed",
				"ERROR transaction b2c3d4e5-f6a7-8901-bcde-f12345678901 failed",
			},
			WantNew:     1,
			WantKnown:   1,
			WantPatterns: 1,
		},
		{
			Name: "JavaStyleStacktrace",
			Lines: []string{
				"WARN [SendWorker:188978561024:QuorumCnxManager$SendWorker@679] Interrupted while waiting",
				"WARN [SendWorker:188978561025:QuorumCnxManager$SendWorker@679] Interrupted while waiting",
			},
			WantNew:     1,
			WantKnown:   1,
			WantPatterns: 1,
		},
		{
			Name: "SingleLine",
			Lines: []string{"ERROR single occurrence"},
			WantNew:     1,
			WantKnown:   0,
			WantPatterns: 1,
		},
		{
			Name:        "Empty",
			Lines:       nil,
			WantNew:     0,
			WantKnown:   0,
			WantPatterns: 0,
		},
	}
}
