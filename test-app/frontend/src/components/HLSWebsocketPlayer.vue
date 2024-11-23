<template>
  <div class="hls-player">
    <video controls></video>
  </div>
</template>

<script lang="ts">
import { HlsWebsocket } from '@asanrom/hls-websocket-cdn';


export default {
  name: "HLSWebsocketPlayer",
  props: {
    cdnUrl: String,
    cdnAuth: String,
    streamId: String,
    latency: Number,
  },
  setup: function () {
    return {
      hls: null as HlsWebsocket | null,
    };
  },
  methods: {
    load: function () {
      if (!this.cdnUrl || !this.cdnAuth) {
        return;
      }

      const video = this.$el.querySelector("video");

      if (!video) {
        return;
      }

      if (!HlsWebsocket.isSupported()) {
        return;
      }

      const hls = new HlsWebsocket({
        cdnServerUrl: this.cdnUrl,
        authToken: this.cdnAuth,
        streamId: this.streamId,
        debug: true,
      }, { enableWorker: false, debug: true, liveMaxLatencyDuration: (this.latency || 60) + 1, liveSyncDuration: this.latency || 60 });
      this.hls = hls;
      hls.start();
      hls.attachMedia(video);

      video.play();
    },

    clear() {
      if (this.hls) {
        this.hls.destroy();
        this.hls = null;
      }

      const video = this.$el.querySelector("video");

      if (!video) {
        return;
      }

      video.pause();
      video.removeAttribute('src'); // empty source
      video.load();
    }
  },
  mounted: function () {
    this.load();
  },
  beforeUnmount: function () {
    this.clear();
  },
  watch: {
    streamId: function () {
      this.clear();
      this.load();
    },
    latency: function () {
      this.clear();
      this.load();
    },
  }
};
</script>
