/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package contractapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ================================
// Tests
// ================================

func TestSetMetadata(t *testing.T) {
	sc := systemContract{}
	sc.setMetadata("my metadata")

	assert.Equal(t, "my metadata", sc.metadata, "should have set metadata field")
}

func TestGetMetadata(t *testing.T) {
	sc := systemContract{}
	sc.metadata = "my metadata"

	assert.Equal(t, "my metadata", sc.GetMetadata(), "should have returned metadata field")
}
