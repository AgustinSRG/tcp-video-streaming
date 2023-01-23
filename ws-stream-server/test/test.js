/* Main test client script */

window.escapeHTML = function escapeHTML(html) {
    return ("" + html).replace(/&/g, "&amp;").replace(/</g, "&lt;")
        .replace(/>/g, "&gt;").replace(/"/g, "&quot;")
        .replace(/'/g, "&apos;");
}

function main() {
    document.getElementById("playButton").addEventListener("click", onPlay);
    document.getElementById("publishButton").addEventListener("click", onPublish);
    document.getElementById("stopButton").addEventListener("click", onStop);

    document.getElementById("main-video").addEventListener("timeupdate", onTimeUpdate);
}

document.addEventListener("DOMContentLoaded", main);

var ws = null;
var recorder = null;
var pubStream = null;

var playChunks = [];
var playBlob = null;

var playCurrentTime = 0;

function onTimeUpdate(ev) {
    var t = document.getElementById("main-video").currentTime;
    if (!isNaN(t) && t > 0) {
        playCurrentTime = document.getElementById("main-video").currentTime;
    }
}

function onPlay() {
    var channel = document.getElementById("channelInput").value;

    if (!channel) {
        document.getElementById("errMsg").innerHTML = "Please, provide a channel ID";
        return;
    }

    var key = document.getElementById("keyInput").value;

    if (!key) {
        document.getElementById("errMsg").innerHTML = "Please, provide a streaming key";
        return;
    }

    document.getElementById("playButton").disabled = true;
    document.getElementById("publishButton").disabled = true;
    document.getElementById("stopButton").disabled = true;
    document.getElementById("errMsg").innerHTML = "";

    console.log("PLAY");

    playChunks = [];
    playCurrentTime = 0;

    document.getElementById("main-video").pause();
    document.getElementById("main-video").removeAttribute('src');
    document.getElementById("main-video").load();

    ws = new WebSocket((location.protocol === "https:" ? "wss:" : "ws:") + "//" + location.hostname + "/" + encodeURIComponent(channel) + "/" + encodeURIComponent(key) + "/receive");

    ws.onopen = function () {
        console.log("Websocket connected!");
    };

    ws.onmessage = function (ev) {
        var data = ev.data;

        if (typeof data === "string") {
            console.log("<<< " + data);
            if (data.startsWith("ERROR:")) {
                document.getElementById("errMsg").innerHTML = escapeHTML(data);
            }
        } else {
            // ArrayBuffer

            console.log(data);

            playChunks.push(data);

            const blob = new Blob(playChunks, { type: "video/webm" });
            const blobURL = window.URL.createObjectURL(blob);

            document.getElementById("main-video").src = blobURL;
            document.getElementById("main-video").currentTime = playCurrentTime;
            document.getElementById("main-video").play();
        }
    };

    ws.onclose = function () {
        console.log("Websocket closed!");
        document.getElementById("playButton").disabled = false;
        document.getElementById("publishButton").disabled = false;
        document.getElementById("stopButton").disabled = true;
        document.getElementById("main-video").pause();
        ws = null;

        if (playBlob) {
            window.URL.revokeObjectURL(playBlob);
            playBlob = null;
        }

        playChunks = [];
    };

    ws.onerror = function (err) {
        console.error(err);
        document.getElementById("errMsg").innerHTML = escapeHTML(err.message);
    };
}

function onPublish() {
    var channel = document.getElementById("channelInput").value;

    if (!channel) {
        document.getElementById("errMsg").innerHTML = "Please, provide a channel ID";
        return;
    }

    var key = document.getElementById("keyInput").value;

    if (!key) {
        document.getElementById("errMsg").innerHTML = "Please, provide a streaming key";
        return;
    }

    document.getElementById("playButton").disabled = true;
    document.getElementById("publishButton").disabled = true;
    document.getElementById("stopButton").disabled = true;
    document.getElementById("errMsg").innerHTML = "";

    console.log("PUBLISH");

    navigator.mediaDevices.getDisplayMedia({
        video: true,
    }).then(stream => {
        pubStream = stream;
        document.getElementById("stopButton").disabled = false;

        document.getElementById("main-video").srcObject = stream;
        document.getElementById("main-video").muted = true;
        document.getElementById("main-video").play();

        ws = new WebSocket((location.protocol === "https:" ? "wss:" : "ws:") + "//" + location.hostname + "/" + encodeURIComponent(channel) + "/" + encodeURIComponent(key) + "/publish");

        ws.onopen = function () {
            console.log("Websocket connected!");

            // Start transmitting the data stream

            recorder = new MediaRecorder(stream, {
                mimeType: "video/webm;codecs=opus, vp8",
            });

            recorder.ondataavailable = function (ev) {
                if (ev.data.size === 0) return;
                if (ws) {
                    ws.send(ev.data)
                } else if (recorder) {
                    recorder.stop();
                }
            };

            recorder.start(3000);
        };

        ws.onmessage = function (ev) {
            var data = ev.data;

            if (typeof data === "string") {
                console.log("<<< " + data);
                if (data.startsWith("ERROR:")) {
                    document.getElementById("errMsg").innerHTML = escapeHTML(data);
                }
            }
        };

        detectStreamEnding(stream, () => {
            onStop();
        });

        ws.onclose = function () {
            console.log("Websocket closed!");
            document.getElementById("playButton").disabled = false;
            document.getElementById("publishButton").disabled = false;
            document.getElementById("stopButton").disabled = true;
            document.getElementById("main-video").srcObject = undefined;
            document.getElementById("main-video").pause();
            ws = null;

            if (recorder) {
                recorder.stop();
                recorder = null;
            }

            if (pubStream) {
                stopStream(pubStream);
                pubStream = null;
            }
        };

        ws.onerror = function (err) {
            console.error(err);
            document.getElementById("errMsg").innerHTML = escapeHTML(err.message);
        };
    }).catch(err => {
        console.error(err);
        document.getElementById("errMsg").innerHTML = escapeHTML(err.message);
        document.getElementById("playButton").disabled = false;
        document.getElementById("publishButton").disabled = false;
        document.getElementById("stopButton").disabled = true;
    });
}

function onStop() {
    if (ws) {
        ws.close();
        ws = null;
    }
    if (recorder) {
        recorder.stop();
        recorder = null;
    }
}

function stopStream(stream) {
    try {
        stream.stop();
    } catch (ex) { }
    try {
        stream.getTracks().forEach(function (track) {
            track.stop();
        });
    } catch (ex) { }
}

function detectStreamEnding(stream, handler) {
    var tracks = stream.getTracks();
    if (tracks.length > 0) {
        tracks[0].addEventListener("ended", handler);
    }
}

setInterval(() => {
    if (ws) {
        ws.send("h");
    }
}, 20000);
