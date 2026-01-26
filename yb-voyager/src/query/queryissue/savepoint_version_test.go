//go:build unit

/*
Copyright (c) YugabyteDB, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package queryissue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yugabyte/yb-voyager/yb-voyager/src/ybversion"
)

// TestSavepointIssueFixedInVersions verifies that the savepoint issue
// is correctly marked as fixed in specific YugabyteDB versions
func TestSavepointIssueFixedInVersions(t *testing.T) {
	sqls := []string{
		`SAVEPOINT my_savepoint;`,
		`BEGIN;
		INSERT INTO accounts (id, balance) VALUES (1, 1000);
		SAVEPOINT sp1;
		UPDATE accounts SET balance = balance - 100 WHERE id = 1;
		ROLLBACK TO SAVEPOINT sp1;
		COMMIT;`,
	}

	// Test with versions where savepoint is NOT fixed (should report issue)
	versionsNotFixed := []*ybversion.YBVersion{
		ybversion.V2024_2_4_0, // Before 2024.2.8.0
		ybversion.V2025_1_0_0, // Before 2025.1.3.0
		ybversion.V2025_2_0_0, // Before 2025.2.1.0
	}

	for _, version := range versionsNotFixed {
		t.Run("NotFixed_"+version.String(), func(t *testing.T) {
			parserIssueDetector := NewParserIssueDetector()
			for _, sql := range sqls {
				issues, err := parserIssueDetector.GetDMLIssues(sql, version)
				assert.NoError(t, err, "Error detecting issues for statement: %s", sql)

				// For these versions, we should see the savepoint issue
				if len(issues) > 0 {
					foundSavepointIssue := false
					for _, issue := range issues {
						if issue.Type == SAVEPOINT_USAGE {
							foundSavepointIssue = true
							break
						}
					}
					assert.True(t, foundSavepointIssue, 
						"Savepoint issue should be reported for version %s", version.String())
				}
			}

			// Verify that savepoint usage was tracked
			assert.True(t, parserIssueDetector.IsSavepointUsed(), 
				"Savepoint usage should be detected for version %s", version.String())
		})
	}

	// Test with versions where savepoint IS fixed (should NOT report issue)
	versionsFixed := []*ybversion.YBVersion{
		ybversion.V2024_2_8_0, // Fixed in 2024.2.8.0
		ybversion.V2025_1_3_0, // Fixed in 2025.1.3.0
		ybversion.V2025_2_1_0, // Fixed in 2025.2.1.0
	}

	for _, version := range versionsFixed {
		t.Run("Fixed_"+version.String(), func(t *testing.T) {
			parserIssueDetector := NewParserIssueDetector()
			for _, sql := range sqls {
				issues, err := parserIssueDetector.GetDMLIssues(sql, version)
				assert.NoError(t, err, "Error detecting issues for statement: %s", sql)

				// For these versions, savepoint issue should be filtered out
				foundSavepointIssue := false
				for _, issue := range issues {
					if issue.Type == SAVEPOINT_USAGE {
						foundSavepointIssue = true
						break
					}
				}
				assert.False(t, foundSavepointIssue, 
					"Savepoint issue should NOT be reported for version %s (fixed)", version.String())
			}

			// Savepoint usage is still tracked, but issue is filtered
			assert.True(t, parserIssueDetector.IsSavepointUsed(), 
				"Savepoint usage should still be detected for version %s", version.String())
		})
	}
}

// TestSavepointVersionConstants verifies that the version constants are correctly defined
func TestSavepointVersionConstants(t *testing.T) {
	// Verify version constants exist and are valid
	assert.NotNil(t, ybversion.V2024_2_8_0, "2024.2.8.0 version should be defined")
	assert.NotNil(t, ybversion.V2025_1_3_0, "2025.1.3.0 version should be defined")
	assert.NotNil(t, ybversion.V2025_2_1_0, "2025.2.1.0 version should be defined")

	// Verify version strings
	assert.Equal(t, "2024.2.8.0", ybversion.V2024_2_8_0.String())
	assert.Equal(t, "2025.1.3.0", ybversion.V2025_1_3_0.String())
	assert.Equal(t, "2025.2.1.0", ybversion.V2025_2_1_0.String())

	// Verify version series
	assert.Equal(t, ybversion.SERIES_2024_2, ybversion.V2024_2_8_0.Series())
	assert.Equal(t, ybversion.SERIES_2025_1, ybversion.V2025_1_3_0.Series())
	assert.Equal(t, ybversion.SERIES_2025_2, ybversion.V2025_2_1_0.Series())

	// Verify version comparisons
	assert.True(t, ybversion.V2024_2_8_0.GreaterThanOrEqual(ybversion.V2024_2_4_0))
	assert.True(t, ybversion.V2025_1_3_0.GreaterThanOrEqual(ybversion.V2025_1_0_0))
	assert.True(t, ybversion.V2025_2_1_0.GreaterThanOrEqual(ybversion.V2025_2_0_0))
}
