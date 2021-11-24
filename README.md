# gorm-firebird
GORM firebird driver

import:

"github.com/flylink888/gorm-firebird"

Example:

```
var products []Product
dsn := "SYSDBA:masterkey@127.0.0.1/sysdb?charset=utf8"
db, err := gorm.Open(firebird.Open(dsn), &gorm.Config{})
if err != nil {
	fmt.Println(err)
}
```

```
type Product struct {
PID  string `gorm:"primaryKey"`
NAME string
}
```

```
func (Product) TableName() string {
return "PRODUCT"
}
```
