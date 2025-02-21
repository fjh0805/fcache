package test

import (
	"log"
	"testing"

	"github.com/limerence-yu/fcache/DB"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var m = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
	"Dick": "1000",
}

func TestCreate(t *testing.T) {
	dsn := "root:1234@tcp(127.0.0.1:3306)/studentdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}

	db.AutoMigrate(&DB.Student{})
	for name, score := range m {
		result := db.Where("name = ?", name).First(&DB.Student{})
		if result.Error == nil {
			log.Printf("Student with name %s already exists!", name)
		} else {
			db.Create(&DB.Student{Name: name, Score: score})
		}
	}
}

func TestDelete(t *testing.T) {
	dsn := "root:123123@tcp(127.0.0.1:3306)/studentdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}
	//软删除 软删除是指数据并不会从数据库中真正删除，而是通过标记（通常是设置一个 deleted_at 字段）来表示该数据已被删除
	db.Where("id in (?)", []int{1}).Delete(&DB.Student{})
	//硬删除 硬删除是指数据从数据库中被完全删除，不会保留任何痕迹。
	db.Where("id in (?)", []int{2}).Unscoped().Delete(&DB.Student{})
}

func TestUpdate(t *testing.T) {
	dsn := "root:123123@tcp(127.0.0.1:3306)/studentdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}
	// update 更改单个字段，updates更改多个字段
	db.Where("id = ?", 1).First(&DB.Student{}).Updates(DB.Student{
		Name:  "John",
		Score: "79",
	})
}

func TestFind(t *testing.T) {
	dsn := "root:123123@tcp(127.0.0.1:3306)/studentdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}
	var student []DB.Student
	//第一种方法
	db.First(&student, "score = ?", "589")
	//第二种方法
	// db.Where("id < ?  ", 5).Find(&student)
	log.Println(student)


	// // 查询包括已软删除的记录
	// db.Unscoped().Find(&student)  // 返回所有记录，包括已软删除的

	// // 查询不包括已软删除的记录（默认行为）
	// db.Find(&student)  // 不返回已软删除的记录
}


func TestRestore(t *testing.T) {
	dsn := "root:123123@tcp(127.0.0.1:3306)/studentdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}
	// 假设我们想恢复 id 为 5 和 6 的学生记录
	db.Unscoped().Model(&DB.Student{}).Where("id in (?)", []int{1, 6}).Update("deleted_at", nil)
}