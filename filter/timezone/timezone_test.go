package filtertimezone

import (
	"fmt"
	"testing"
	"time"
)

func Test_Filter(t *testing.T) {
	//ev := utils.LogEvent{}
	locat, err := time.LoadLocation("asia/shand ghai")
	fmt.Println(locat, err)
	t1, err := time.ParseInLocation("2006-01-02", "2017-03-05", locat)
	fmt.Println(t1, err)
	zone, offset := t1.Zone()
	fmt.Println(zone, offset)
}
