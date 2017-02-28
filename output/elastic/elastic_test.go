package outputelastic

import (
	"testing"
	"zonst/tuhuayuan/logagent/utils"

	"fmt"

	elastic "gopkg.in/olivere/elastic.v5"
)

func init() {
	utils.RegistOutputHandler(PluginName, InitHandler)
}

func Test_Init(t *testing.T) {
	_, err := elastic.NewClient()
	if err != nil {
		fmt.Println(err)
	}
}
