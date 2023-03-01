<template>
  <div class="hls-player">
    <video controls></video>
  </div>
</template>

<script lang="ts">

import Hls from "hls.js/dist/hls.min";

export default {
  name: "HLSPlayer",
  props: {
    url: String,
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
        const hls = new Hls();
        this.$options.hls = hls;
        hls.loadSource(this.url);
        hls.attachMedia(video);
      } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
        video.src = this.url;
      }

      video.play();
    },

    clear() {
      if (this.$options.hls) {
        this.$options.hls.destroy();
        this.$options.hls = null;
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
    url: function () {
      this.clear();
      this.load();
    },
  }
};
</script>
