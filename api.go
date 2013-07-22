package main

import (
	"database/sql"
	"errors"
	"github.com/astaxie/beedb"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

func getUsersInCompanySkill(company, skill string) (error, [10]string) {
	var users [10]string
	users[0] = "PeterSZW,Skill"
	return nil, users
	//return errors.New("TODO"), nil
}
func getUsersInCompany(company string) (error, [10]string) {
	var users [10]string
	users[0] = "PeterSZW"
	return nil, users
	//return errors.New("TODO"), nil
}

func companyNeedSkill(company string, skill string) error {
	return errors.New("TODO")
}

func companyAdd(company string) error {
	return errors.New("TODO")
}
func companyUpdate(company string, companyInfo string) error {
	return errors.New("TODO")
}
func companyDelete(company string) error {
	return errors.New("TODO")
}

func userHasNoSkill(username string, skill string) error {
	return errors.New("TODO")
}

func userJoinCompany(username string, Company string) error {
	return errors.New("TODO")
}

func userLeftCompany(username string, Company string) error {
	return errors.New("TODO")
}

func userHasSkill(username string, skill string) error {
	return errors.New("TODO")

}
func userLogin(username string, password string) error {

	var one Userinfo
	orm.Where("Username=?", username).Find(&one)

	if one.Username != "" {
		if one.Password == password {
			return nil
		} else {
			return errors.New("Password is not corect")
		}
	} else {
		return errors.New("No user " + username)
	}

}
func userSignup(username string, password string) error {

	var one Userinfo
	orm.Where("Username=?", username).Find(&one)

	if one.Username == "" {
		var saveone Userinfo
		saveone.Username = username
		saveone.Password = password
		saveone.Departname = ""
		saveone.Created = time.Now().Format("2006-01-02 15:04:05")
		orm.Save(&saveone)
		return nil
	} else {
		return errors.New("Username is Exisit " + username)
	}

}

func userChangePassword(username string, oldpassword string, newpassword string) error {

	var one Userinfo
	orm.Where("Username=?", username).Find(&one)

	if one.Username != "" {
		if one.Password == oldpassword {

			//original SQL update

			t := make(map[string]interface{})
			t["password"] = newpassword
			orm.SetTable("userinfo").SetPK("uid").Where(one.Uid).Update(t)

			return nil
		} else {
			return errors.New("Password is not corect")
		}
	} else {
		return errors.New("No user " + username)
	}

}

func loadDB() {
	db, err := sql.Open("sqlite3", "./httpserver.s3db")
	if err != nil {
		panic(err)
	}
	orm = beedb.New(db)

}
