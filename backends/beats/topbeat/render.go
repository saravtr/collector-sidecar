// This file is part of Graylog.
//
// Graylog is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Graylog is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Graylog.  If not, see <http://www.gnu.org/licenses/>.

package topbeat

import (
	"os/exec"
	"io/ioutil"
	"bytes"
	"text/template"

	"github.com/Graylog2/collector-sidecar/api/graylog"
	"github.com/Graylog2/collector-sidecar/util"
)

func (tbc *TopBeatConfig) snippetsToString() string {
	var buffer bytes.Buffer
	var result bytes.Buffer
	for _, snippet := range tbc.Beats.Snippets {
		snippetTemplate, err := template.New("snippet").Parse(snippet.Value)
		if err != nil {
			result.WriteString(snippet.Value)
		} else {
			snippetTemplate.Execute(&buffer, tbc.Beats.Context.Inventory)
			result.WriteString(buffer.String())
		}
		result.WriteString("\n")
	}
	return result.String()
}

func (tbc *TopBeatConfig) Render() bytes.Buffer {
	var result bytes.Buffer

	if tbc.Beats.Data() == nil {
		return result
	}

	result.WriteString(tbc.Beats.String())
	result.WriteString(tbc.snippetsToString())

	return result
}

func (tbc *TopBeatConfig) RenderToFile(path string) error {
	stringConfig := tbc.Render()
	err := util.CreatePathToFile(path)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, stringConfig.Bytes(), 0644)
	return err
}

func (tbc *TopBeatConfig) RenderOnChange(response graylog.ResponseCollectorConfiguration, path string) bool {
	newConfig := NewCollectorConfig(tbc.Beats.Context)

	for _, output := range response.Outputs {
		if output.Backend == "topbeat" {
			for property, value := range output.Properties {
				newConfig.Beats.Set(value, "output", output.Type, property)
			}
		}
	}
	for _, input := range response.Inputs {
		if input.Backend == "topbeat" {
			for property, value := range input.Properties {
				newConfig.Beats.Set(value, "input", property)
			}
		}
	}
	for _, snippet := range response.Snippets {
		if snippet.Backend == "topbeat" {
			newConfig.Beats.AppendString(snippet.Id, snippet.Value)
		}
	}

	if !tbc.Beats.Equals(newConfig.Beats) {
		log.Info("Configuration change detected, rewriting configuration file.")
		tbc.Beats.Update(newConfig.Beats)
		tbc.RenderToFile(path)
		return true
	}

	return false
}

func (tbc *TopBeatConfig) ValidateConfigurationFile(configurationPath string) bool {
	cmd := exec.Command(tbc.ExecPath(), "-configtest", "-c", configurationPath)
	err := cmd.Run()
	if err != nil {
		log.Error("Error during configuration validation: ", err)
		return false
	}

	return true
}