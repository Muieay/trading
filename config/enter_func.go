package config

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

/** 输入辅助函数 **/

func InputString(reader *bufio.Reader, prompt, defaultValue string) string {
	fmt.Printf("%s [默认: %s]: ", prompt, defaultValue)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func InputInt(reader *bufio.Reader, prompt string, defaultValue int) int {
	fmt.Printf("%s [默认: %d]: ", prompt, defaultValue)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(input)
	if err != nil {
		fmt.Printf("⚠️  输入无效，使用默认值: %d\n", defaultValue)
		return defaultValue
	}
	return value
}

func InputFloat(reader *bufio.Reader, prompt string, defaultValue float64) float64 {
	fmt.Printf("%s [默认: %.4f]: ", prompt, defaultValue)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(input, 64)
	if err != nil {
		fmt.Printf("⚠️  输入无效，使用默认值: %.4f\n", defaultValue)
		return defaultValue
	}
	return value
}

func InputBool(reader *bufio.Reader, prompt string, defaultValue bool) bool {
	defaultStr := "N"
	if defaultValue {
		defaultStr = "Y"
	}
	fmt.Printf("%s (y/N) [默认: %s]: ", prompt, defaultStr)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultValue
	}
	return input == "y" || input == "yes"
}
