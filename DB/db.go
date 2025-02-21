package DB

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 将 Name 字段映射到数据库中的 VARCHAR(255) 类型。
// 为该字段创建一个名为 idx_name 的索引。
type Student struct {
	gorm.Model
	Name string `json:"name" gorm:"type:varchar(255);index:idx_name"`
	Score string `json:"score"`
}

func Init() (*gorm.DB, error) {
	dsn := "root:1234@tcp(127.0.0.1:3306)/studentdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}

	db.AutoMigrate(&Student{})
	return db, nil
}
