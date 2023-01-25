// Publisher methods for a session

package main

import (
	"container/list"
	"crypto/subtle"
)

// Starts a specific player
// Call only for publishers
// player - The player session
func (session *WS_Streaming_Session) StartPlayer(player *WS_Streaming_Session) {
	session.publishMutex.Lock()
	defer session.publishMutex.Unlock()

	if !session.isPublishing {
		player.isPlaying = false
		player.isIdling = true
		player.log("PLAY IDLE '" + player.channel + "'")
		return
	}

	player.log("PLAY START '" + player.channel + "'")

	if !player.gopPlayNo && session.gopCache.Len() > 0 {
		for t := session.gopCache.Front(); t != nil; t = t.Next() {
			chunks := t.Value
			switch x := chunks.(type) {
			case *DataStreamChunk:
				player.SendChunk(x.data)
			}
		}
	}

	player.isPlaying = true
	player.isIdling = false

	if player.gopPlayClear {
		session.gopCache = list.New()
		session.gopCacheSize = 0
		session.gopCacheDisabled = true
	}
}

// Starts sending to idle players
// Call only for publishers
func (session *WS_Streaming_Session) StartIdlePlayers() {
	session.publishMutex.Lock()
	defer session.publishMutex.Unlock()

	// Start idle players
	idlePlayers := session.server.GetIdlePlayers(session.channel)

	for i := 0; i < len(idlePlayers); i++ {
		player := idlePlayers[i]
		if subtle.ConstantTimeCompare([]byte(session.key), []byte(player.key)) == 1 {
			player.log("PLAY START '" + player.channel + "'")

			if !player.gopPlayNo && session.gopCache.Len() > 0 {
				for t := session.gopCache.Front(); t != nil; t = t.Next() {
					chunks := t.Value
					switch x := chunks.(type) {
					case *DataStreamChunk:
						player.SendChunk(x.data)
					}
				}
			}

			player.isPlaying = true
			player.isIdling = false

			if player.gopPlayClear {
				session.gopCache = list.New()
				session.gopCacheSize = 0
				session.gopCacheDisabled = true
			}
		} else {
			player.log("Error: Invalid stream key provided")
			player.SendText("ERROR: Invalid streaming key")
			player.Kill()
		}
	}
}

// Handles a data stream chunk
// data - The chunk data
func (session *WS_Streaming_Session) HandleChunk(data []byte) {
	chunkLength := len(data)
	chunk := DataStreamChunk{
		data: data,
		size: chunkLength,
	}

	session.publishMutex.Lock()
	defer session.publishMutex.Unlock()

	if !session.isPublishing {
		return
	}

	// GOP cache

	if !session.gopCacheDisabled {
		session.gopCache.PushBack(&chunk)
		session.gopCacheSize += int64(chunkLength) + DATA_STREAM_PACKET_BASE_SIZE

		for session.gopCacheSize > session.gopCacheLimit {
			toDelete := session.gopCache.Front()
			v := toDelete.Value
			switch x := v.(type) {
			case *DataStreamChunk:
				session.gopCacheSize -= int64(x.size)
			}
			session.gopCache.Remove(toDelete)
			session.gopCacheSize -= DATA_STREAM_PACKET_BASE_SIZE
		}
	}

	// Players

	players := session.server.GetPlayers(session.channel)

	for i := 0; i < len(players); i++ {
		if players[i].isPlaying {
			players[i].SendChunk(data)
		}
	}
}

// Finishes a publishing session
// Call only for publishers
func (session *WS_Streaming_Session) EndPublish() {
	session.publishMutex.Lock()
	defer session.publishMutex.Unlock()

	if session.isPublishing {

		session.log("PUBLISH END '" + session.channel + "'")

		players := session.server.GetPlayers(session.channel)

		for i := 0; i < len(players); i++ {
			players[i].isIdling = true
			players[i].isPlaying = false
			players[i].log("PLAY END '" + players[i].channel + "'")
			players[i].Kill()
		}

		session.server.RemovePublisher(session.channel)

		session.gopCache = list.New()

		session.isPublishing = false

		// Send event
		if session.server.controlConnection.PublishEnd(session.channel, session.streamId) {
			session.debug("Stop event sent")
		} else {
			session.debug("Could not send stop event")
		}
	}
}
