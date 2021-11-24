# gorm-firebird
GORM firebird driver

import:

"github.com/flylink888/gorm-firebird"

Example:

```
var products []Product
dsn := "SYSDBA:masterkey@127.0.0.1/sysdb?charset=utf8"
db, err := gorm.Open(firebird.Open(dsn), &gorm.Config{
   NamingStrategy: firebird.NamingStrategy{},  //这个很重要
})
if err != nil {
	fmt.Println(err)
}
```

```
type Product struct {
Pid  string `gorm:"primaryKey"`
Name string
}
```

```
//这个不是必需的
func (Product) TableName() string {
return "PRODUCT"
}
```
