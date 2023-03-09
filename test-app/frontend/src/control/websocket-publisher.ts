// Websocket publisher

"use strict";

export declare interface WebSocketPublisher {
    addEventListener(event: 'open', listener: (ev: Event) => void): this;
    addEventListener(event: 'close', listener: (ev: Event) => void): this;
    addEventListener(event: 'error', listener: (ev: ErrorEvent) => void): this;
}

const RECORDER_INTERVAL = 500;
const HEARTBEAT_INTERVAL = 20 * 1000;

function stopStream(stream: MediaStream) {
    try {
        stream.getTracks().forEach(function (track) {
            track.stop();
        });
    } catch (ex) { }
}

export function getWebsocketPublishingURL(base: string, channel: string, key: string) {
    return base + "/" + channel + "/" + key + "/publish";
}

export class WebSocketPublisher extends EventTarget {
    public ws: WebSocket;

    public stream: MediaStream;

    public connected: boolean;

    public heartBeatTimer: number | undefined;

    public recorder: MediaRecorder | null;

    constructor(stream: MediaStream, url: string) {
        super();
        this.ws = new WebSocket(url);
        this.stream = stream;
        this.recorder = null;

        const tracks = stream.getTracks();
        if (tracks.length > 0) {
            tracks[0].addEventListener("ended", this.onStreamEnded.bind(this));
        }

        this.connected = false;

        this.ws.onopen = this.onOpen.bind(this);
        this.ws.onerror = this.onError.bind(this);
        this.ws.onclose = this.onClose.bind(this);
        this.ws.onmessage = this.onMessage.bind(this);

        this.heartBeatTimer = undefined;
    }

    private onOpen() {
        this.connected = true;

        this.recorder = new MediaRecorder(this.stream);
        this.recorder.ondataavailable = this.onRecorderData.bind(this);
        this.recorder.start(RECORDER_INTERVAL);

        this.heartBeatTimer = setInterval(this.sendHeartBeat.bind(this), HEARTBEAT_INTERVAL);

        this.dispatchEvent(new Event("open"));
    }

    private onRecorderData(ev: BlobEvent) {
        if (!this.connected || !this.recorder) {
            return;
        }

        if (ev.data.size === 0) return;

        this.ws.send(ev.data); // Send media chunk
    }

    private sendHeartBeat() {
        if (!this.connected) {
            return;
        }

        this.ws.send("h");
    }

    private onClose() {
        this.connected = false;
        if (this.heartBeatTimer !== undefined) {
            clearInterval(this.heartBeatTimer);
            this.heartBeatTimer = undefined;
        }

        if (this.recorder) {
            this.recorder.stop();
            this.recorder = null;
        }

        stopStream(this.stream);

        this.dispatchEvent(new Event("close"));
    }

    private onError(err: Event) {
        this.dispatchEvent(new ErrorEvent('error', { error: err }));
    }


    private onMessage(ev: MessageEvent) {
        var data = ev.data;

        if (typeof data === "string") {
            if (data.startsWith("ERROR:")) {
                this.dispatchEvent(new ErrorEvent('error', { error: new Error(data) }));
            }
        }
    }

    private onStreamEnded() {
        this.close();
    }

    public close() {
        this.ws.close();

        if (this.recorder) {
            this.recorder.stop();
            this.recorder = null;
        }

        stopStream(this.stream);
    }
}
