package main

import (
	"errors"
	"strings"
	"time"
)

type Options struct {
	InstanceId   string `json:"instance_id" long:"instance-id" description:"the instance to be collected" required:"true"`
	InspectionId string `json:"inspection_id" long:"inspection-id" description:"a unique id to identity this inspection" required:"true"`
	Home         string `json:"home" long:"home" description:"foresight working directory" required:"true"`
	Items        string `json:"collect" long:"items" description:"items to collect" required:"true"`
	Begin        string `json:"begin" long:"begin" description:"scrape begin time"`
	End          string `json:"end" long:"end" description:"scrape begin time"`
	Components   string `json:"components" long:"components" description:"components to be profile"`
}

func (o *Options) GetInstanceId() string {
	return o.InstanceId
}

func (o *Options) GetInspectionId() string {
	return o.InspectionId
}

func (o *Options) GetHome() string {
	return o.Home
}

func (o *Options) GetItems() []string {
	return strings.Split(o.Items, ",")
}

func (o *Options) GetScrapeBegin() (time.Time, error) {
	if o.Begin == "" {
		return time.Time{}, errors.New("begin time not specified in command line")
	}
	return time.Parse(time.RFC3339, o.Begin)
}

func (o *Options) GetScrapeEnd() (time.Time, error) {
	if o.End == "" {
		return time.Time{}, errors.New("end time not specified in command line")
	}
	return time.Parse(time.RFC3339, o.End)
}

func (o *Options) GetComponents() []string {
	return strings.Split(o.Components, ",")
}
