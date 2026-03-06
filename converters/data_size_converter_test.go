// Copyright 2024-2025 NetCracker Technology Corporation
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package converters

import "testing"

func TestConvertToBytes(t *testing.T) {
	table := []struct {
		input  string
		output uint64
	}{
		{"MB", 0},
		{"1MB", 1048576},
		{"-2MB", 0},
		{"4MB", 4194304},
		{"6 MB", 0},
		{"sevenMB", 0},
		{"GB", 0},
		{"1GB", 1073741824},
		{"-2GB", 0},
		{"4GB", 4294967296},
		{"6 GB", 0},
		{"sevenGB", 0},
		{"1TB", 0}, // Invalid unit
	}
	for _, tableValue := range table {
		value, _ := ToBytes(tableValue.input)
		if value != tableValue.output {
			t.Errorf("Unexpected value. Expected => (%d), Received => (%d)", tableValue.output, value)
		}
	}
}
