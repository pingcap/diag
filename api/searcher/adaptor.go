package searcher

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type ItemType int16

const (
	TypeTiDB ItemType = iota
	TypeTiKV
	TypePD
	TypeTiDBSlowQuery
)

type LevelType int16

const (
	LevelFATAL LevelType = iota
	LevelERROR
	LevelWARN
	LevelINFO
	LevelDEBUG
)

const (
	TimeStampLayout       = "2006/01/02 15:04:05.000 -07:00"
	FormerTimeStampLayout = "2006/01/02 15:04:05.000"
)

var (
	TiDBLogRE = regexp.MustCompile(`^\[([^\[\]]*)\]\s\[([^\[\]]*)\]`)
	TiKVLogRE = regexp.MustCompile(`^([0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3})\s([^\s]*)`)
	PDLogRE   = regexp.MustCompile(`^([0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3})\s[^\s]*\s\[([^\[\]]*)\]`)
)

type TidbLogParser struct {
	BaseParser
}

func (p *TidbLogParser) Next() (err error) {
	defer func() {
		if err != nil {
			p.current = nil
		}
	}()
	line, err := p.BaseParser.Next()
	if err != nil {
		return err
	}
	// TODO: TiDB version < 2.1.8, using unified log format
	matches := TiDBLogRE.FindStringSubmatch(line)
	if len(matches) < 3 {
		return errors.New("failed to parser tidb log:" + line)
	}
	t := matches[1]
	ts, err := parseTimeStamp(t)
	if err != nil {
		return err
	}
	level, err := parseLogLevel(matches[2])
	if err != nil {
		return err
	}
	l := p.NewLogItem(line, ts, level)
	l.Type = TypeTiDB
	p.current = l
	return nil
}

type TikvLogParser struct {
	BaseParser
}

func (p *TikvLogParser) Next() (err error) {
	defer func() {
		if err != nil {
			p.current = nil
		}
	}()

	line, err := p.BaseParser.Next()
	if err != nil {
		return err
	}
	// TODO: TiKV version >= 2.1.15 or >= 3.0.0, using former log format
	matches := TiKVLogRE.FindStringSubmatch(line)
	if len(matches) < 3 {
		log.Warnf("skip parse unsupported line:" + line)
		return p.Next()
	}
	ts, err := parseFormerTimeStamp(matches[1])
	if err != nil {
		fmt.Println(line)
		return err
	}
	level, err := parseLogLevel(matches[2])
	if err != nil {
		return err
	}
	l := p.NewLogItem(line, ts, level)
	l.Type = TypeTiKV
	p.current = l
	return nil
}

type PDLogParser struct {
	BaseParser
}

func (p *PDLogParser) Next() (err error) {
	defer func() {
		if err != nil {
			p.current = nil
		}
	}()

	line, err := p.BaseParser.Next()
	if err != nil {
		return err
	}
	// TODO: PD will use the unified log format
	// PD has not implemented unified log format at present (2019/07/19).
	matches := PDLogRE.FindStringSubmatch(line)
	if len(matches) < 3 {
		log.Warnf("skip parse unsupported line:" + line)
		return p.Next()
	}
	ts, err := parseFormerTimeStamp(matches[1])
	if err != nil {
		fmt.Println(line)
		return err
	}
	level, err := parseLogLevel(matches[2])
	if err != nil {
		return err
	}
	l := p.NewLogItem(line, ts, level)
	l.Type = TypePD
	p.current = l
	return nil
}

type TidbSlogQueryParser struct {
	BaseParser
}

func (p *TidbSlogQueryParser) Next() (err error) {
	defer func() {
		if err != nil {
			p.current = nil
		}
	}()
	var ts *time.Time
	var content string
	for {
		byts, _, err := p.Reader.ReadLine()
		if err != nil {
			return err
		}
		line := string(byts)
		//log.Infof("tidb slow query line:%s\n", line)
		if !strings.HasPrefix(line, "#") &&
			bytes.Contains(byts, p.SearchBytes) {
			content = line
			break
		}
		t := p.parseTime(line)
		if t != nil {
			ts = t
		}
	}
	l := p.NewLogItem(content, ts, -1)
	l.Type = TypeTiDBSlowQuery
	p.current = l
	return nil
}

func (p *TidbSlogQueryParser) parseTime(line string) *time.Time {
	re := regexp.MustCompile("^# Time: (.*)$")
	if !re.MatchString(line) {
		return nil
	}
	m := re.FindStringSubmatch(line)
	t, e := time.Parse(time.RFC3339, m[1])
	if e != nil {
		return nil
	}
	return &t
}

// TiDB / TiKV / PD unified log format
// [2019/03/04 17:04:24.614 +08:00] ...
func parseTimeStamp(str string) (*time.Time, error) {
	t, err := time.Parse(TimeStampLayout, str)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TiDB / TiKV / PD log format used in former version
// 2019/07/18 11:04:29.314 ...
func parseFormerTimeStamp(content string) (*time.Time, error) {
	local, err := time.LoadLocation("Asia/Chongqing")
	if err != nil {
		return nil, err
	}
	t, err := time.ParseInLocation(FormerTimeStampLayout, content, local)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

var LevelTypeMap = map[string]LevelType{
	"FATAL": LevelFATAL,
	"ERROR": LevelERROR,
	"WARN":  LevelWARN,
	"INFO":  LevelINFO,
	"DEBUG": LevelDEBUG,
}

func parseLogLevel(s string) (LevelType, error) {
	s = strings.ToUpper(s)
	if s == "ERRO" {
		return LevelERROR, nil
	}
	if s == "CRITICAL" {
		return LevelFATAL, nil
	}
	if s == "WARNING" {
		return LevelWARN, nil
	}
	if level, ok := LevelTypeMap[s]; ok {
		return level, nil
	} else {
		return -1, errors.New("failed to parse log level: " + s)
	}
}
