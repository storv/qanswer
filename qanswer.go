package qanswer

import (
	"regexp"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/ngaut/log"
	termbox "github.com/nsf/termbox-go"
	"github.com/silenceper/qanswer/config"
	"github.com/silenceper/qanswer/proto"
	"github.com/silenceper/qanswer/util"
)

//Run start run
func Run() {

	cfg := config.GetConfig()
	err := util.MkDirIfNotExist(proto.ImagePath)
	if err != nil {
		panic(err)
	}
	err = termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	if !cfg.Debug {
		log.SetLevel(log.LOG_LEVEL_INFO)
	}

	color.Cyan("基本配置：")
	color.Cyan("平台：%s; 图片识别方式：%s", cfg.Device, cfg.OcrType)
	color.Yellow("\n请按空格键开始搜索答案：")

Loop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeySpace:
				answerQuestion(cfg)
				color.Yellow("\n\n请按空格键开始搜索答案：")
			default:
				break Loop
			}
		}
	}

}

func answerQuestion(cfg *config.Config) {
	color.Cyan("正在开始搜索....")
	//区分ios 或android 获取图像
	screenshot := NewScreenshot(cfg)
	png, err := screenshot.GetImage()
	if err != nil {
		log.Errorf("获取截图失败，%v", err)
		return
	}
	err = saveImage(png, cfg)
	if err != nil {
		log.Errorf("保存图片失败，%v", err)
		return
	}

	//识别文字
	ocr := NewOcr(cfg)
	var wg sync.WaitGroup
	wg.Add(2)

	var questionText string
	go func() {
		defer wg.Done()
		questionText, err = ocr.GetText(proto.QuestionImage)
		if err != nil {
			log.Errorf("识别题目失败，%v", err)
			return
		}
		questionText = processQuestion(questionText)
	}()

	var answerArr []string
	go func() {
		defer wg.Done()
		answerText, err := ocr.GetText(proto.AnswerImage)
		if err != nil {
			log.Errorf("识别答案失败，%v", err)
			return
		}
		answerArr = processAnswer(answerText)
	}()
	wg.Wait()

	if cfg.Debug {
		color.Yellow("识别题目：")
		color.Green("%s", questionText)
		color.Yellow("识别答案：")
		color.Green("%v", answerArr)
	}

	//搜索答案并显示
	result := GetSearchResult(questionText, answerArr)
	for engine, answerResult := range result {
		color.Yellow("\n%s的搜索结果:", engine)
		color.Cyan("题目：%s \n", questionText)
		for key, val := range answerResult {
			color.Green("%s : 结果数 %d ， 答案出现频率： %d", answerArr[key], val.Sum, val.Freq)
		}
	}
}

func processQuestion(text string) string {
	log.Debug(text)
	text = strings.Replace(text, "\n", "", -1)
	text = strings.Replace(text, "\r", "", -1)

	//去除编号
	re, _ := regexp.Compile("\\d\\.")
	text = re.ReplaceAllString(text, "")
	return text
}

func processAnswer(text string) []string {
	log.Debug(text)
	text = strings.Replace(text, " ", "", -1)
	arr := strings.Split(text, "\n")
	return arr
}
