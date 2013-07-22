package main

import (
	"testing"
	"time"
)

func Test_Benchmark_getAdmin(t *testing.T) {
	loadDB()
	orm.SetTable("userinfo").Where("uid>?", 0).DeleteRow()

	var saveone Userinfo
	saveone.Username = "PeterSZW"
	saveone.Password = "1234"
	saveone.Departname = "Test Add Departname"
	saveone.Created = time.Now().Format("2006-01-02 15:04:05")
	orm.Save(&saveone)

	err := userLogin("PeterSZW", "1234")
	if err != nil {
		t.Error("userLogin Error")
	}
	err2 := userLogin("PeterSZW", "123")
	if err2 == nil {
		t.Error("userLogin Error")
	}

	userChangePassword("PeterSZW", "1234", "123")

	if userLogin("PeterSZW", "123") != nil {
		t.Error("userChangePassword Error")
	}
	//
}
