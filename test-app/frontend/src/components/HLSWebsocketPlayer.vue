<template>
  <div class="hls-player">
    <video controls @ended="onEnded" @waiting="onStalled" @playing="onPlaying"></video>
  </div>
</template>

<script lang="ts">
import { useVModel } from '@/utils/vmodel';
import { HlsWebSocket } from '@asanrom/hls-websocket-cdn';

export default {
  name: "HLSWebsocketPlayer",
  props: {
    cdnUrl: String,
    cdnAuth: String,
    streamId: String,
    latency: Number,
    stalled: Boolean,
  },
  emits: ['ended', 'update:stalled'],
  setup: function (props) {
    return {
      hls: null as HlsWebSocket | null,

      stalledStatus: useVModel(props, "stalled"),
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
        delay: (this.latency || 60) - 1,
        maxDelay: this.latency || 60,
      });
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

    onStalled: function () {
      this.stalledStatus = true;
    },

    onPlaying: function () {
      this.stalledStatus = false;
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
    latency: function (newValue: number, oldValue: number) {
      if (newValue === oldValue) {
        return;
      }

      if (newValue < oldValue) {
        if (this.hls) {
          this.hls.setDelayOptions(newValue - 1, newValue);
        }
      } else {
        this.clear();
        this.load();
      }
    },
  }
};
</script>
