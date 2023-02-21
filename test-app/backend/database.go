// JSON database to store streaming app status

package main

import (
	"crypto/subtle"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const DB_TMP_FILE = "database.json.tmp"
const DB_FILE = "database.json"

var DATABASE *StreamingTestAppDatabase

type StreamingTestAppDatabase struct {
	mutex *sync.Mutex

	data StreamingTestAppData

	writing            bool
	pendingWrite       bool
	pendingDataToWrite []byte
}

type StreamingTestAppData struct {
	Channels map[string]*StreamingChannel `json:"channels"`
}

type StreamingChannel struct {
	Id  string `json:"id"`
	Key string `json:"key"`

	Record      bool   `json:"record"`
	Resolutions string `json:"resolutions"`
	Previews    string `json:"previews"`

	Live     bool   `json:"live"`
	StreamId string `json:"streamId"`

	LiveStartTimestamp int64 `json:"liveStartTimestamp"`

	LiveSubStreams []SubStream `json:"liveSubStreams"`

	VODList []VODStreaming `json:"vodList"`
}

type VODStreaming struct {
	StreamId      string      `json:"streamId"`
	Timestamp     int64       `json:"timestamp"`
	SubStreams    []SubStream `json:"subStreams"`
	HasPreviews   bool        `json:"hasPreviews"`
	PreviewsIndex string      `json:"previewsIndex"`
}

type SubStream struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	FPS       int    `json:"fps"`
	IndexFile string `json:"indexFile"`
}

func CreateStreamingTestAppDatabase() *StreamingTestAppDatabase {
	db := &StreamingTestAppDatabase{
		mutex:              &sync.Mutex{},
		writing:            false,
		pendingWrite:       false,
		pendingDataToWrite: nil,
	}

	fullPath := DB_FILE

	if os.Getenv("DB_PATH") != "" {
		fullPath = filepath.Join(os.Getenv("DB_PATH"), fullPath)
	}

	data := StreamingTestAppData{}

	content, err := ioutil.ReadFile(fullPath)

	if err == nil {
		err = json.Unmarshal(content, &data)

		if err != nil {
			data = StreamingTestAppData{}
		}
	}

	if data.Channels == nil {
		data.Channels = make(map[string]*StreamingChannel)
	}

	db.data = data

	return db
}

func (db *StreamingTestAppDatabase) Save() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	data, err := json.Marshal(db.data)

	if err != nil {
		return err
	}

	if db.writing {
		db.pendingWrite = true
		db.pendingDataToWrite = data
	} else {
		db.writing = true
		go db.writeToFile(data)
	}

	return nil
}

const (
	FILE_PERMISSION   = 0600 // Read/Write
	FOLDER_PERMISSION = 0700 // Read/Write/Run
)

func (db *StreamingTestAppDatabase) writeToFile(data []byte) {
	fullPath := DB_FILE

	if os.Getenv("DB_PATH") != "" {
		fullPath = filepath.Join(os.Getenv("DB_PATH"), fullPath)
	}

	tmpPath := fullPath + ".tmp"

	done := false
	dataToWrite := data

	for !done {
		err := ioutil.WriteFile(tmpPath, dataToWrite, FILE_PERMISSION)

		if err != nil {
			LogError(err)
		} else {
			LogDebug("JSON database saved!")
		}

		db.mutex.Lock()

		if db.pendingWrite {
			dataToWrite = db.pendingDataToWrite
			db.pendingWrite = false
			db.pendingDataToWrite = nil
		} else {
			db.writing = false
			done = true
		}

		db.mutex.Unlock()
	}
}

func (db *StreamingTestAppDatabase) VerifyKey(channel string, key string) (valid bool, record bool, previews string, resolutions string) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	channelData := db.data.Channels[channel]

	if channelData == nil {
		return false, false, "", ""
	}

	if subtle.ConstantTimeCompare([]byte(key), []byte(channelData.Key)) != 1 {
		LogDebug("Invalid key!")
		return false, false, "", ""
	}

	return true, channelData.Record, channelData.Previews, channelData.Resolutions
}

func (channel *StreamingChannel) FindOrCreateVOD(streamId string) *VODStreaming {
	for i := 0; i < len(channel.VODList); i++ {
		if channel.VODList[i].StreamId == streamId {
			if channel.VODList[i].SubStreams == nil {
				channel.VODList[i].SubStreams = make([]SubStream, 0)
			}

			return &channel.VODList[i]
		}
	}

	newStreaming := VODStreaming{
		StreamId:      streamId,
		Timestamp:     time.Now().UnixMilli(),
		SubStreams:    make([]SubStream, 0),
		HasPreviews:   false,
		PreviewsIndex: "",
	}

	channel.VODList = append(channel.VODList, newStreaming)

	return &channel.VODList[len(channel.VODList)-1]
}

func (db *StreamingTestAppDatabase) AddAvailableStream(channel string, streamId string, streamType string, resolution string, indexFile string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	channelData := db.data.Channels[channel]

	if channelData == nil {
		return nil
	}

	if channelData.LiveSubStreams == nil {
		channelData.LiveSubStreams = make([]SubStream, 0)
	}

	if channelData.VODList == nil {
		channelData.VODList = make([]VODStreaming, 0)
	}

	parsedResolution, err := DecodeResolution(resolution)

	if err != nil {
		return err
	}

	subStream := SubStream{
		IndexFile: indexFile,
		Width:     parsedResolution.width,
		Height:    parsedResolution.height,
		FPS:       parsedResolution.fps,
	}

	if streamType == "HLS-LIVE" {
		channelData.Live = true
		if streamId != channelData.StreamId {
			channelData.StreamId = streamId
			channelData.LiveSubStreams = make([]SubStream, 0)
		}

		alreadyExists := false

		for i := 0; i < len(channelData.LiveSubStreams); i++ {
			if channelData.LiveSubStreams[i].IndexFile == indexFile {
				alreadyExists = true
				break
			}
		}

		if !alreadyExists {
			channelData.LiveSubStreams = append(channelData.LiveSubStreams, subStream)
		}
	} else if streamType == "HLS-VOD" {
		vod := channelData.FindOrCreateVOD(streamId)

		alreadyExists := false

		for i := 0; i < len(vod.SubStreams); i++ {
			if vod.SubStreams[i].IndexFile == indexFile {
				alreadyExists = true
				break
			}
		}

		if !alreadyExists {
			vod.SubStreams = append(vod.SubStreams, subStream)
		}
	} else if streamType == "IMG-PREVIEW" {
		vod := channelData.FindOrCreateVOD(streamId)

		vod.HasPreviews = true
		vod.PreviewsIndex = indexFile
	}

	return nil
}

func (db *StreamingTestAppDatabase) CloseStream(channel string, streamId string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	channelData := db.data.Channels[channel]

	if channelData == nil {
		return nil
	}

	if channelData.Live && channelData.StreamId == streamId {
		channelData.Live = false
		channelData.StreamId = ""
	}

	return nil
}
