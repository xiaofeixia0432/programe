package calldb

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

var Db *sql.DB

func Mysql_init(user, password, ip, dbname string, port int) {
	//?multiStatements=true
	var dsn string
	var err error
	dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?multiStatements=true", user, password, ip, port, dbname)
	//dsn := user + ":" + password + "@tcp(" + ip + ":" + strconv.Itoa(port) + ")/" + dbname + "?multiStatements=true"
	Db, err = sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("Open database error: %s\n", err)
	}
	Db.SetMaxOpenConns(2000)
	Db.SetMaxIdleConns(1000)
	err = Db.Ping()
	if err != nil {
		fmt.Println(err)
	}
}

func Call_procedure(procedure_name, param_in string) (int, string, int, string, error) {

	/*call procedure */
	exec_sql := fmt.Sprintf("call %s('%s',@out1,@out2,@out3,@out4)", procedure_name, param_in)
	_, err := Db.Exec(exec_sql)
	//_, err := db.Exec(`call wlas_gettime('',@out1,@out2,@out3,@out4)`)
	if err != nil {
		fmt.Println(err)
	}

	/* select out parameter */
	rows, err := Db.Query(`select @out1,@out2,@out3,@out4`)
	if err != nil {
		fmt.Println(err)
	}
	var out_ret int
	var out_ansStr string
	var out_sqlCode int
	var out_errMsg string
	for rows.Next() {

		rows.Scan(&out_ret, &out_ansStr, &out_sqlCode, &out_errMsg)
		//fmt.Println(out_ret, out_ansStr, out_sqlCode, out_errMsg)
	}
	return out_ret, out_ansStr, out_sqlCode, out_errMsg, err
}
