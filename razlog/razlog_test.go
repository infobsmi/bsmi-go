package razlog

import (
	"log"
	"testing"
)

func TestAndOperate(t *testing.T) {
	var razlog = NewRazLog(3)
	razlog.Add("hello world0").Do(log.Println)
	razlog.Add("hello world1").Do(log.Println)
	razlog.Add("hello world2").Do(log.Println)
	razlog.Add("hello world3").Do(log.Println)
	razlog.Add("hello world4").Do(log.Println)
	razlog.Add("hello world5").Do(log.Println)
	razlog.Add("hello world6").Do(log.Println)
	razlog.Add("hello world7").Do(log.Println)
	razlog.Add("hello world8").Do(log.Println)
	//
}
