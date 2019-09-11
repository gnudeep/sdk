/*
 * Copyright (c) 2019 WSO2 Inc. (http:www.wso2.org) All Rights Reserved.
 *
 * WSO2 Inc. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http:www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 *
 */

package kubectl

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
)

func getCommandOutput(cmd *exec.Cmd) (string, error) {
	exitCode := 0
	var output string
	stdoutReader, _ := cmd.StdoutPipe()
	stdoutScanner := bufio.NewScanner(stdoutReader)
	go func() {
		for stdoutScanner.Scan() {
			output += stdoutScanner.Text()
		}
	}()
	stderrReader, _ := cmd.StderrPipe()
	stderrScanner := bufio.NewScanner(stderrReader)
	go func() {
		for stderrScanner.Scan() {
			output += stderrScanner.Text()
		}
	}()
	err := cmd.Start()
	if err != nil {
		return output, err
	}
	err = cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exit, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = exit.ExitStatus()
			}
			if exitCode == 1 {
				return output, fmt.Errorf(output)
			}
		}
		return output, err
	}
	return output, nil
}

func getCommandOutputFromTextFile(cmd *exec.Cmd) ([]byte, error) {
	exitCode := 0
	var output string
	outfile, err := os.Create("./out.txt")
	if err != nil {
		return nil, err
	}
	defer outfile.Close()
	cmd.Stdout = outfile
	stderrReader, _ := cmd.StderrPipe()
	stderrScanner := bufio.NewScanner(stderrReader)
	go func() {
		for stderrScanner.Scan() {
			output += stderrScanner.Text()
		}
	}()
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	err = cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exit, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = exit.ExitStatus()
			}
			if exitCode == 1 {
				return nil, fmt.Errorf(output)
			}
		}
		return nil, err
	}
	out, err := ioutil.ReadFile("./out.txt")
	os.Remove("./out.txt")
	return out, err
}
