// Copyright 2022 Linkall Inc.
//
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

package internal

import cdkgo "github.com/vanus-labs/cdk-go"

var _ cdkgo.SourceConfigAccessor = &GitHubAPIConfig{}

type GitHubAPIConfig struct {
	cdkgo.SourceConfig `json:",inline" yaml:",inline"`
	Organizations      []string `json:"organizations" yaml:"organizations" validate:"required"`
	GitHubAccessToken  string   `json:"github_access_token" yaml:"github_access_token" validate:"required"`
	GitHubHourLimit    int      `json:"github_hour_limit" yaml:"github_hour_limit"`
}

func NewConfig() cdkgo.SourceConfigAccessor {
	return &GitHubAPIConfig{}
}
