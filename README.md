# gorm-firebird
GORM firebird driver

Example:

`var products []Product
dsn := "SYSDBA:masterkey@127.0.0.1/sysdb?charset=none"
db, err := gorm.Open(firebird.Open(dsn), &gorm.Config{})
if err != nil {
	fmt.Println(err)
}`

`type Product struct {
PID  string ``gorm:"primaryKey"``
NAME string
}`

`func (Product) TableName() string {
return "PRODUCT"
}
`