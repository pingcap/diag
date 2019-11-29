package osconfig

import "io/ioutil"

func cater(fileName string) string {
	datas, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(err)
	}
	return string(datas)
}
