package fileLogger

import (
	"fmt"
	config "github.com/resssoft/tgbot-template/configuration"
	"github.com/resssoft/tgbot-template/internal/mediator"
	"github.com/resssoft/tgbot-template/internal/models"
	"github.com/rs/zerolog/log"
	"os"
	"sync"
	"time"
)

const (
	beforeFileCloseDuration = 1 * time.Second
	statusCheckDuration     = 5 * time.Second
	chanBufferSize          = 10000
)

type Application interface {
	CloseAll()
	Log(string, string, bool, bool)
	AddSource(string, string) error
}

type logFile struct {
	file    *os.File
	path    string
	channel chan string
}

type Client struct {
	mapSafety *sync.Mutex
	File      map[string]logFile
}

func Provide(dispatcher *mediator.Dispatcher) Application {
	client := &Client{
		File:      make(map[string]logFile),
		mapSafety: &sync.Mutex{},
	}

	if err := dispatcher.Register(
		Listener{
			Client: client,
		},
		models.FileLoggerEvents...); err != nil {
		log.Info().Err(err).Send()
	}
	go client.checkStatus()
	return client
}

func (c *Client) CloseAll() {
	for _, fileLog := range c.File {
		close(fileLog.channel)
	}
	time.Sleep(beforeFileCloseDuration)
	for fileName, fileLog := range c.File {
		err := fileLog.file.Close()
		if err != nil {
			log.Info().Err(err).Msgf("close file %s error", fileName)
		}
	}
}

func (c *Client) Log(src string, data string, withoutTime bool, toDebug bool) {
	row := ""
	c.mapSafety.Lock()
	file := c.File[src]
	c.mapSafety.Unlock()
	if withoutTime {
		row = data
	} else {
		row = fmt.Sprintf("[%s] %s \n", time.Now().Format(config.DateTimeFormat), data)
	}
	file.channel <- row
	if toDebug {
		log.Debug().Msg(data)
	}
}

func (c *Client) AddSource(filePath, name string) error {
	file, err := os.OpenFile(config.LogPath()+filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	c.mapSafety.Lock()
	c.File[name] = logFile{
		file:    file,
		path:    config.LogPath() + filePath,
		channel: make(chan string, chanBufferSize),
	}
	go c.worker(c.File[name])
	c.mapSafety.Unlock()
	return nil
}

func (c *Client) worker(fileLog logFile) {
	log.Info().Msgf("Start fileLogger worker for file: %s", fileLog.path)
	for data := range fileLog.channel {
		_, err := fileLog.file.WriteString(data)
		if err != nil {
			log.Info().Err(err).Msgf("File %s write error", fileLog.path)
		}
	}
}

func (c *Client) checkStatus() {
	for {
		time.Sleep(statusCheckDuration)
		c.mapSafety.Lock()
		for _, fileLog := range c.File {
			count := len(fileLog.channel)
			if count != 0 {
				log.Info().Msgf("The queue: %v", count)
			}
			switch {
			case count > 10:
				log.Info().Msgf("The queue is big enough: %v", count)
			case count > 900:
				log.Warn().Msgf("The queue starts to fill: %v", count)
			}
		}
		c.mapSafety.Unlock()
	}
}
