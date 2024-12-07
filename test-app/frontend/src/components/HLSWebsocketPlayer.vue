<template>
  <div class="hls-player">
    <video controls @ended="onEnded"></video>
  </div>
</template>

<script lang="ts">
import { HlsWebSocket } from '@asanrom/hls-websocket-cdn';


export default {
  name: "HLSWebsocketPlayer",
  props: {
    cdnUrl: String,
    cdnAuth: String,
    streamId: String,
    latency: Number,
  },
  emits: ['ended'],
  setup: function () {
    return {
      hls: null as HlsWebSocket | null,
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

      if (!HlsWebSocket.isSupported()) {
        return;
      }

      const hls = new HlsWebSocket({
        cdnServerUrl: this.cdnUrl,
        authToken: this.cdnAuth,
        streamId: this.streamId,
        debug: true,
      }, { debug: true, liveMaxLatencyDuration: (this.latency || 60) + 1, liveSyncDuration: this.latency || 60 });
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
    },

    onEnded: function () {
      this.$emit("ended");
    },
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
