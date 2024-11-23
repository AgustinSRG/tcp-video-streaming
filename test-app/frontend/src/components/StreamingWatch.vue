<template>
  <div class="main-content">
    <div v-if="found">
      <h2>Channel: #{{ channelId }}</h2>
      <div class="channel-watch-group">
        <h3 v-if="!live">Status: Offline</h3>
        <h3 v-if="live">Status: Live</h3>
        <p v-if="live">Live time: {{ renderTime(liveNow - liveStartTimestamp) }}</p>
        <div class="form-group" v-if="live">
          <select class="form-control" v-model="selectedLiveSubStream" @change="onUpdatedCdn()">
            <option :value="''">-- Select a quality to play it --</option>
            <option v-for="ss in liveSubStreams" :key="ss.indexFile" :value="ss.indexFile">{{ ss.width }}x{{ ss.height }},
              {{ ss.fps }} fps</option>
          </select>
        </div>
        <div class="form-group" v-if="live && hasCdnSupport(selectedLiveSubStream, liveSubStreams)" @change="onUpdatedCdn()">
          <input v-model="preferCdn" type="checkbox" value="prefer-cdn">
          <label>Use HLS Websocket CDN?</label>
        </div>
        <div class="" v-if="isCdn">
          <HLSWebsocketPlayer :cdn-url="cdnUrl" :cdn-auth="cdnAuth" :stream-id="selectedLiveSubStream"></HLSWebsocketPlayer>
        </div>
        <div class="" v-else>
          <HLSPlayer :url="getHLSURL(selectedLiveSubStream, liveSubStreams)"></HLSPlayer>
        </div>
      </div>

      <hr />

      <p v-if="vods.length === 0">There are no VODs available for this channel.</p>

      <p v-if="vods.length > 0">List of available VODs:</p>
      <ul v-if="vods.length > 0">
        <li v-for="vod in vods">[{{ renderDate(vod.timestamp) }}] <RouterLink :to="'/watch/' + channelId + '/vod/' + vod.streamId">./vod/{{ vod.streamId }}</RouterLink>
        </li>
      </ul>
    </div>
    <div v-if="!found">
      <h2>Channel not found</h2>
    </div>
  </div>
</template>

<script lang="ts">
import { type SubStreamWithCdn, WatchAPI, type ChannelStatus, type VODItem, type VODItemList } from "@/api/api-watch";

import HLSPlayer from "./HLSPlayer.vue";
import HLSWebsocketPlayer from "./HLSWebsocketPlayer.vue";
import ConfirmationModal from "./ConfirmationModal.vue";
import { RouterLink } from 'vue-router';
import { GetAssetURL, Request } from "@/utils/request";
import { renderTimeSeconds } from "@/utils/time-utils";
import { ChannelStorage } from "@/control/channel-storage";
import { Timeouts } from "@/utils/timeout";
import { HlsWebsocket } from "@asanrom/hls-websocket-cdn";

interface ComponentData {
  found: boolean;
  channelId: string;
  channelKey: string;

  live: boolean;
  liveStartTimestamp: number;
  liveNow: number;
  liveSubStreams: SubStreamWithCdn[];
  selectedLiveSubStream: string,

  preferCdn: boolean,

  isCdn: boolean,
  cdnUrl: string,
  cdnAuth: string,

  vods: VODItem[];
}

export default {
  name: "StreamingControl",
  emits: [],
  components: {
    HLSPlayer,
    HLSWebsocketPlayer,
    RouterLink,
    ConfirmationModal,
  },
  setup: function () {
    return {
      tickTimer: 0,
    };
  },
  data: function (): ComponentData {
    return {
      found: false,
      channelId: "",
      channelKey: "",

      live: false,
      liveStartTimestamp: 0,
      liveSubStreams: [],
      liveNow: Date.now(),
      selectedLiveSubStream: "",

      preferCdn: true,

      isCdn: false,
      cdnAuth: "",
      cdnUrl: "",

      vods: [],
    };
  },
  methods: {
    getHLSURL: function (selectedLiveSubStream: string, liveSubStreams: SubStreamWithCdn[]): string {
      for (let ss of liveSubStreams) {
        if (ss.indexFile === selectedLiveSubStream) {
          return GetAssetURL("/" + ss.indexFile);
        }
      }

      return "";
    },

    getCdnStreamId: function (selectedLiveSubStream: string, liveSubStreams: SubStreamWithCdn[]): string {
      for (let ss of liveSubStreams) {
        if (ss.indexFile === selectedLiveSubStream) {
          return ss.indexFile;
        }
      }

      return "";
    },

    hasCdnSupport: function (selectedLiveSubStream: string, liveSubStreams: SubStreamWithCdn[]): boolean {
      if (!HlsWebsocket.isSupported()) {
        return false;
      }

      for (let ss of liveSubStreams) {
        if (ss.indexFile === selectedLiveSubStream) {
          return !!ss.cdnUrl && !!ss.cdnAuth;
        }
      }

      return false;
    },

    canWatchByCdn: function (selectedLiveSubStream: string, liveSubStreams: SubStreamWithCdn[], preferCdn: boolean): boolean {
      if (!this.hasCdnSupport(selectedLiveSubStream, liveSubStreams)) {
        return false;
      }

      return preferCdn;
    },

    onUpdatedCdn: function () {
      this.isCdn = this.canWatchByCdn(this.selectedLiveSubStream, this.liveSubStreams, this.preferCdn);

      for (let ss of this.liveSubStreams) {
        if (ss.indexFile === this.selectedLiveSubStream) {
          this.cdnUrl = ss.cdnUrl;
          this.cdnAuth = ss.cdnAuth;
        }
      }
    },

    updateNow: function () {
      this.liveNow = Date.now();
    },

    renderTime: function (t: number): string {
      return renderTimeSeconds(Math.round(t / 1000))
    },

    renderDate: function (t: number): string {
      return (new Date(t)).toISOString();
    },

    autoSelectLiveStream: function () {
      for (let ss of this.liveSubStreams) {
        if (ss.indexFile === this.selectedLiveSubStream) {
          return;
        }
      }

      if (this.liveSubStreams.length > 0) {
        this.selectedLiveSubStream = this.liveSubStreams[0].indexFile;
      } else {
        this.selectedLiveSubStream = "";
      }

      this.onUpdatedCdn();
    },

    findChannel: function () {
      const channel = ChannelStorage.GetChannel(this.$route.params.channel + "");

      this.channelId = this.$route.params.channel + "";

      if (!channel) {
        this.channelKey = "";
      } else {
        this.channelKey = channel.key;
      }

      this.live = false;
      this.loadChannelStatus();
      this.loadChannelVODList();
    },

    loadChannelStatus: function () {
      Timeouts.Abort("load-channel-status-watch");

      Request.Pending("load-channel-status-watch",
        WatchAPI.GetChannelStatus(this.channelId)
      )
        .onSuccess((result: ChannelStatus) => {
          this.found = true;
          this.live = result.live;
          this.liveStartTimestamp = result.liveStartTimestamp;
          this.liveSubStreams = result.liveSubStreams.sort((a, b) => {
            const aSize = a.width * a.height;
            const bSize = b.width * b.height;

            if (aSize > bSize) {
              return -1;
            } else if (bSize > aSize) {
              return 1;
            } else if (a.fps > b.fps) {
              return -1;
            } else {
              return 1;
            }
          });
          this.autoSelectLiveStream();
          Timeouts.Set("load-channel-status-watch", 5000, this.loadChannelStatus.bind(this));
        })
        .onRequestError((err) => {
          Request.ErrorHandler()
            .add("404", "*", () => {
              this.found = false;
            })
            .add("*", "*", () => {
              Timeouts.Set("load-channel-status-watch", 2000, this.loadChannelStatus.bind(this));
            })
            .handle(err);
        })
        .onUnexpectedError((err) => {
          console.error(err);
          Timeouts.Set("load-channel-status-watch", 2000, this.loadChannelStatus.bind(this));
        });
    },

    loadChannelVODList: function () {
      Timeouts.Abort("load-channel-vods-watch");

      Request.Pending("load-channel-vods-watch",
        WatchAPI.GetChannelVODList(this.channelId)
      )
        .onSuccess((result: VODItemList) => {
          this.found = true;
          this.vods = result.vod_list.sort((a, b) => {
            if (a.timestamp > b.timestamp) {
              return -1;
            } else {
              return 1;
            }
          });
          Timeouts.Set("load-channel-vods-watch", 5000, this.loadChannelVODList.bind(this));
        })
        .onRequestError((err) => {
          Request.ErrorHandler()
            .add("404", "*", () => {
              return;
            })
            .add("*", "*", () => {
              Timeouts.Set("load-channel-vods-watch", 2000, this.loadChannelVODList.bind(this));
            })
            .handle(err);
        })
        .onUnexpectedError((err) => {
          console.error(err);
          Timeouts.Set("load-channel-vods-watch", 2000, this.loadChannelVODList.bind(this));
        });
    },
  },
  mounted: function () {
    this.findChannel();

    this.tickTimer = setInterval(this.updateNow.bind(this), 500);
  },
  beforeUnmount: function () {
    Timeouts.Abort("load-channel-status-watch");
    Request.Abort("load-channel-status-watch");

    Timeouts.Abort("load-channel-vods-watch");
    Request.Abort("load-channel-vods-watch");

    clearInterval(this.tickTimer);
  },
  watch: {
    $route() {
      this.findChannel();
    }
  },
};
</script>