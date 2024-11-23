<template>
  <div class="hls-player">
    <video controls @ended="onEnded"></video>
  </div>
</template>

<script lang="ts">

import Hls from "hls.js";

export default {
  name: "HLSPlayer",
  props: {
    url: String,
    latency: Number,
  },
  emits: ['ended'],
  setup: function () {
    return {
      hls: null as Hls | null,
    };
  },
  methods: {
    load: function () {
      if (!this.url) {
        return;
      }

      const video = this.$el.querySelector("video");

      if (!video) {
        return;
      }

      if (Hls.isSupported()) {
        const hls = new Hls({ enableWorker: false, liveMaxLatencyDuration: (this.latency || 60) + 1, liveSyncDuration: this.latency || 60 });
        this.hls = hls;
        hls.loadSource(this.url);
        hls.attachMedia(video);
      } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
        video.src = this.url;
      }

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
    url: function () {
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
