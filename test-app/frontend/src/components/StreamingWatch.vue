<template>
  <div class="main-content">
    <div v-if="found">
      <h2>Channel: #{{ channelId }}</h2>
      <div class="channel-watch-group">
        <h3 v-if="!live">Status: Offline</h3>
        <h3 v-if="live">Status: Live</h3>
        <p v-if="live">Live time: {{ renderTime(liveNow - liveStartTimestamp) }}</p>
        <div class="form-group" v-if="live">
          <select class="form-control" v-model="selectedLiveSubStream">
            <option :value="''">-- Select a quality to play it --</option>
            <option v-for="ss in liveSubStreams" :key="ss.indexFile" :value="ss.indexFile">{{ ss.width }}x{{ ss.height }},
              {{ ss.fps }} fps</option>
          </select>
        </div>
        <div class="">
          <HLSPlayer :url="getHLSURL(selectedLiveSubStream, liveSubStreams)"></HLSPlayer>
        </div>
      </div>

      <hr />

      <p v-if="vods.length === 0">There are no VODs available for this channel.</p>

      <p v-if="vods.length > 0">List of available VODs:</p>
      <ul v-if="vods.length > 0">
        <li v-for="vod in vods">[{{ renderDate(vod.timestamp) }}] <RouterLink to="/">./vod/{{ vod.streamId }}</RouterLink></li>
      </ul>
    </div>
    <div v-if="!found">
      <h2>Channel not found</h2>
    </div>
  </div>
</template>
  
<script lang="ts">
import { WatchAPI, type ChannelStatus, type SubStream, type VODItem, type VODItemList } from "@/api/api-watch";

import HLSPlayer from "./HLSPlayer.vue";
import ConfirmationModal from "./ConfirmationModal.vue";
import { RouterLink } from 'vue-router';
import { GetAssetURL, Request } from "@/utils/request";
import { renderTimeSeconds } from "@/utils/time-utils";
import { ChannelStorage } from "@/control/channel-storage";
import { Timeouts } from "@/utils/timeout";

interface ComponentData {
  found: boolean;
  channelId: string;
  channelKey: string;

  live: boolean;
  liveStartTimestamp: number;
  liveNow: number;
  liveSubStreams: SubStream[];
  selectedLiveSubStream: string,

  vods: VODItem[];
}

export default {
  name: "StreamingControl",
  emits: [],
  components: {
    HLSPlayer,
    RouterLink,
    ConfirmationModal,
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

      vods: [],
    };
  },
  methods: {
    getHLSURL: function (selectedLiveSubStream: string, liveSubStreams: SubStream[]) {
      for (let ss of liveSubStreams) {
        if (ss.indexFile === selectedLiveSubStream) {
          return GetAssetURL("/" + ss.indexFile);
        }
      }

      return "";
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

    this.$options.tickTimer = setInterval(this.updateNow.bind(this), 500);
  },
  beforeUnmount: function () {
    Timeouts.Abort("load-channel-status-watch");
    Request.Abort("load-channel-status-watch");

    Timeouts.Abort("load-channel-vods-watch");
    Request.Abort("load-channel-vods-watch");

    clearInterval(this.$options.tickTimer);
  },
  watch: {
    $route() {
      this.findChannel();
    }
  },
};
</script>