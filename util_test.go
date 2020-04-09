package main

import (
	"encoding/hex"
	"math/big"
	"testing"
)

func TestLeftShift(t *testing.T) {
	if "25.893180161173005034" != LeftShift("25893180161173005034", 18) {
		t.Error("LeftShift('25893180161173005034', 18) is wrong")
	}
	if "25893180161173005.034" != LeftShift("25893180161173005034", 3) {
		t.Error("LeftShift('25893180161173005034', 3) is wrong")
	}
	if "100998995000000000000.126" != LeftShift("100998995000000000000126", 3) {
		t.Error("LeftShift('100998995000000000000126', 3) is wrong")
	}
	if "100998.995000000000000126" != LeftShift("100998995000000000000126", 18) {
		t.Error("LeftShift('100998995000000000000126', 18) is wrong")
	}
	if "0.010099899500000000" != LeftShift("10099899500000000", 18) {
		t.Error("LeftShift('10099899500000000', 18) is wrong")
	}
	if "0.000000000000000000" != LeftShift("0", 18) {
		t.Error("LeftShift('0', 18) is wrong")
	}
	if "0.1234343455555" != LeftShift("12343434.55555", 8) {
		t.Error("LeftShift('12343434.55555', 8) is wrong")
	}
	if "0.001234343455555" != LeftShift("12343434.55555", 10) {
		t.Error("LeftShift('12343434.55555', 10) is wrong")
	}
	if "1234.343455555" != LeftShift("12343434.55555", 4) {
		t.Error("LeftShift('12343434.55555', 4) is wrong")
	}
}

func TestRightShift(t *testing.T) {
	if "0.000005" != RightShift("0.00000000000005", 8) {
		t.Error("RightShift('0.00000000000005', 8) is wrong")
	}
	if "5000000000000000" != RightShift("0.005", 18) {
		t.Error("RightShift('0.005', 8) is wrong")
	}
	if "598390.3883" != RightShift("0.005983903883", 8) {
		t.Error("RightShift('0.005983903883', 8) is wrong")
	}
	if "111133300590000" != RightShift("1111333.0059", 8) {
		t.Error("RightShift('1111333.0059', 8) is wrong")
	}
	if "111100000000" != RightShift("1111", 8) {
		t.Error("RightShift('1111', 8) is wrong")
	}
	if "0.0005" != RightShift("0.0000005", 3) {
		t.Error("RightShift('0.0000005', 3) is wrong")
	}
	if "0.0005" != RightShift("0.0000005000", 3) {
		t.Error("RightShift('0.0000005000', 3) is wrong")
	}
	if "50" != RightShift("0.0000005", 8) {
		t.Error("RightShift('0.0000005', 8) is wrong")
	}
	if "512300000" != RightShift("5.123", 8) {
		t.Error("RightShift('5.123', 8) is wrong")
	}
	if "000000" != RightShift("0.0", 5) {
		t.Error("RightShift('0.0', 5) is wrong")
	}
	if "5123" != RightShift("5.1230", 3) {
		t.Error("RightShift('5.1230', 3) is wrong")
	}
	if "299000000" != RightShift("299.00000000", 6) {
		t.Error("RightShift('299.00000000', 6) is wrong")
	}
}
