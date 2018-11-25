/*
	这个文件是用来根据经纬度获取实际地理位置信息的
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/tealeg/xlsx"
	yaml "gopkg.in/yaml.v1"
)

var (
	dataPath   = ""
	outputPath = ""
	prefix     = "http://api.map.baidu.com/geocoder/v2/?callback=renderReverse&location="
	postfix    = "&output=json&pois=0&ak="
)

type config struct {
	ConfigPath string `yaml:"filepath"`
	Outputfile string `yaml:"outputfile"`
	Ak         string `yaml:"ak"`
}

type ResultData struct {
	Formatted_address   string `json:"formatted_address"`
	Business            string `json:"business"`
	Sematic_description string `json:"sematic_description"`
	CityCode            int    `json:"cityCode"`
}

type Response struct {
	Status int         `json:"status"`
	Result *ResultData `json:"result"`
}

func parseConfig() error {
	cfg := &config{}
	file, err := os.Open("config.yaml")
	if err != nil {
		log.Println("open config file error", err.Error())
		return err
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println("read config content error", err.Error())
		return err
	}
	err = yaml.Unmarshal(content, cfg)
	if err != nil {
		log.Println("yaml unmarshal error", err.Error())
		return err
	}
	dataPath = cfg.ConfigPath
	outputPath = cfg.Outputfile
	postfix += cfg.Ak

	log.Println("parse config ok")
	return nil
}

func getPositionString(lang, lati string) string {
	url := prefix + lati + "," + lang + postfix
	resp, err := http.Get(url)
	if err != nil {
		log.Println("get position once error", err.Error())
		return ""
	}
	defer resp.Body.Close()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("read http body error", err.Error())
		return ""
	}
	first := strings.Index(string(bs), "(")
	length := len(bs)
	js := bs[first+1 : length-1]
	return string(js)
}

func getPosition(lang, lati float64) (string, string, string) {
	//strconv.ParseFloat64(
	lan := strconv.FormatFloat(lang, 'f', 6, 64)
	lat := strconv.FormatFloat(lati, 'f', 6, 64)
	url := prefix + lat + "," + lan + postfix
	resp, err := http.Get(url)
	if err != nil {
		log.Println("get position once error", err.Error())
		return "", "", ""
	}
	defer resp.Body.Close()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("read http body error", err.Error())
		return "", "", ""
	}
	return lan, lat, string(bs)
}

// start_lang,start_lati,end_lang,end_lati,start_pos,end_pos
func addTitle(file *xlsx.File, sheet *xlsx.Sheet) {
	row := sheet.AddRow()
	cell := row.AddCell()
	cell.Value = "start_lang"
	cell = row.AddCell()
	cell.Value = "start_lati"
	cell = row.AddCell()
	cell.Value = "end_lang"
	cell = row.AddCell()
	cell.Value = "end_lati"
	cell = row.AddCell()
	cell.Value = "start_pos"
	cell = row.AddCell()
	cell.Value = "end_pos"
}

func addData(sheet *xlsx.Sheet, sLang, sLati, eLang, eLati, sPos, ePos string) {
	row := sheet.AddRow()
	cell := row.AddCell()
	cell.Value = sLang
	cell = row.AddCell()
	cell.Value = sLati
	cell = row.AddCell()
	cell.Value = eLang
	cell = row.AddCell()
	cell.Value = eLati
	cell = row.AddCell()
	cell.Value = sPos
	cell = row.AddCell()
	cell.Value = ePos
}

func main() {
	var err error
	var file *xlsx.File
	var sheet *xlsx.Sheet

	err = parseConfig()
	if err != nil {
		log.Println("parse config file error", err.Error())
		os.Exit(-1)
	}
	xlFile, err := xlsx.OpenFile(dataPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
	file = xlsx.NewFile()
	sheet, err = file.AddSheet("位置信息")
	if err != nil {
		log.Println("Create Sheet error", err.Error())
		os.Exit(-1)
	}
	addTitle(file, sheet)

	for _, tmpSheet := range xlFile.Sheets {
		if tmpSheet.Name == "字段" {
			continue
		}
		for i, row := range tmpSheet.Rows {
			if i == 0 {
				continue
			}
			sflng := row.Cells[4].String()
			sflat := row.Cells[5].String()
			eflng := row.Cells[6].String()
			eflat := row.Cells[7].String()
			sval := getPositionString(sflng, sflat)
			eval := getPositionString(eflng, eflat)
			sresp := &Response{}
			err = json.Unmarshal([]byte(sval), sresp)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			eresp := &Response{}
			err = json.Unmarshal([]byte(eval), eresp)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			sPos := sresp.Result.Formatted_address + sresp.Result.Sematic_description
			ePos := eresp.Result.Formatted_address + eresp.Result.Sematic_description
			addData(sheet, sflng, sflat, eflng, eflat, sPos, ePos)
		}
	}
	err = file.Save(outputPath)
	if err != nil {
		log.Println("save file error", err.Error())
		os.Exit(-1)
	}
}
