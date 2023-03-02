<template>
  <div class="main-content">
    <div v-if="found">
      <h2>Channel: <RouterLink :to="'/watch/' + channelId">#{{ channelId }}</RouterLink></h2>
      <h3>VOD ID: {{ streamId }}</h3>
      <div class="channel-watch-group">
        <p >VOD date: {{ renderDate(date) }}</p>
        <div class="form-group">
          <select class="form-control" v-model="selectedSubStream">
            <option v-for="ss in subStreams" :key="ss.indexFile" :value="ss.indexFile">VOD[{{ getVODIndex(ss.indexFile) }}] - {{ ss.width }}x{{ ss.height }},
              {{ ss.fps }} fps</option>
          </select>
        </div>
        <div class="">
          <HLSPlayer :url="getHLSURL(selectedSubStream, subStreams)"></HLSPlayer>
        </div>
      </div>

      <hr />

      <p v-if="!hasPreviews">There are no image previews for this VOD.</p>
      <ImagePreviewsViewer v-if="hasPreviews" :url="getPreviewsURL(previewsIndex)"></ImagePreviewsViewer>

      <hr v-if="channelKey" />

      <div v-if="channelKey" class="form-group">
        <div class="form-error" v-if="channelDangerError">{{ channelDangerError }}</div>
        <button type="button" class="btn btn-danger" :disabled="busy" @click="askDelete">Delete</button>
      </div>
      
    </div>
    <div v-if="!found">
      <h2>VOD not found</h2>
    </div>

    <ConfirmationModal v-model:shown="displayConfirmDelete" message="Delete this VOD? (This action is not reversible)"
      @confirm="doDelete"></ConfirmationModal>
  </div>
</template>
  
<script lang="ts">
import { WatchAPI, type SubStream, type VODStreaming } from "@/api/api-watch";

import HLSPlayer from "./HLSPlayer.vue";
import ImagePreviewsViewer from "./ImagePreviewsViewer.vue";
import ConfirmationModal from "./ConfirmationModal.vue";
import { RouterLink } from 'vue-router';
import { GetAssetURL, Request } from "@/utils/request";
import { renderTimeSeconds } from "@/utils/time-utils";
import { ChannelStorage } from "@/control/channel-storage";
import { Timeouts } from "@/utils/timeout";
import { ControlAPI } from "@/api/api-control";
import router from "@/router";
import { getVODIndex } from "@/utils/resolutions";

interface ComponentData {
  found: boolean;
  channelId: string;
  channelKey: string;

  streamId: string;

  date: number;
  subStreams: SubStream[];
  selectedSubStream: string,

  hasPreviews: boolean,
  previewsIndex: string,

  displayConfirmDelete: boolean;

  busy: boolean;

  channelDangerError: string;
}

export default {
  name: "StreamingWatchVOD",
  emits: [],
  components: {
    HLSPlayer,
    RouterLink,
    ConfirmationModal,
    ImagePreviewsViewer,
  },
  data: function (): ComponentData {
    return {
      found: false,
      channelId: "",
      channelKey: "",

      streamId: "",

      date: 0,
      subStreams: [],
      selectedSubStream: "",

      hasPreviews: false,
      previewsIndex: "",

      displayConfirmDelete: false,

      busy: false,

      channelDangerError: "",
    };
  },
  methods: {
    getHLSURL: function (selectedSubStream: string, subStreams: SubStream[]) {
      for (let ss of subStreams) {
        if (ss.indexFile === selectedSubStream) {
          return GetAssetURL("/" + ss.indexFile);
        }
      }

      return "";
    },

    getVODIndex: function (index: string): string {
      return getVODIndex(index) + "";
    },

    getPreviewsURL: function (index: string): string {
      if (!index) {
        return "";
      }
      return GetAssetURL("/" + index);
    },

    renderTime: function (t: number): string {
      return renderTimeSeconds(Math.round(t / 1000))
    },

    renderDate: function (t: number): string {
      return (new Date(t)).toISOString();
    },

    autoSelectSubStream: function () {
      for (let ss of this.subStreams) {
        if (ss.indexFile === this.selectedSubStream) {
          return;
        }
      }

      if (this.subStreams.length > 0) {
        this.selectedSubStream = this.subStreams[0].indexFile;
      } else {
        this.selectedSubStream = "";
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

      this.streamId = this.$route.params.vod + "";

      this.loadVOD();
    },

    loadVOD: function () {
      Timeouts.Abort("load-vod-info");

      Request.Pending("load-vod-info",
        WatchAPI.GetChannelVOD(this.channelId, this.streamId)
      )
        .onSuccess((result: VODStreaming) => {
          this.found = true;
          this.date = result.timestamp;
          this.hasPreviews = result.hasPreviews;
          this.previewsIndex = result.previewsIndex;
          this.subStreams = result.subStreams.sort((a, b) => {
            const aIndex = getVODIndex(a.indexFile);
            const bIndex = getVODIndex(b.indexFile);

            if (aIndex < bIndex) {
              return -1;
            } else if (bIndex < aIndex) {
              return 1;
            }

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
          this.autoSelectSubStream();
        })
        .onRequestError((err) => {
          Request.ErrorHandler()
            .add("404", "*", () => {
              this.found = false;
            })
            .add("*", "*", () => {
              Timeouts.Set("load-vod-info", 2000, this.loadVOD.bind(this));
            })
            .handle(err);
        })
        .onUnexpectedError((err) => {
          console.error(err);
          Timeouts.Set("load-vod-info", 2000, this.loadVOD.bind(this));
        });
    },

    askDelete: function () {
      this.displayConfirmDelete = true;
    },

    doDelete: function () {
      if (this.busy) {
        return;
      }

      this.busy = true;
      this.channelDangerError = "";

      Request.Do(
        ControlAPI.DeleteVOD(this.channelId, this.streamId, this.channelKey)
      )
        .onSuccess(() => {
          this.busy = false;
          router.push('/watch/' + this.channelId);
        })
        .onCancel(() => {
          this.busy = false;
        })
        .onRequestError((err) => {
          this.busy = false;
          Request.ErrorHandler()
            .add(400, "*", () => {
              this.channelDangerError = "Bad request";
            })
            .add(403, "*", () => {
              router.push('/watch/' + this.channelId);
            })
            .add(404, "*", () => {
              router.push('/watch/' + this.channelId);
            })
            .add(500, "*", () => {
              this.channelDangerError = "Internal server error";
            })
            .add("*", "*", () => {
              this.channelDangerError = "Could not connect to the server";
            })
            .handle(err);
        })
        .onUnexpectedError((err) => {
          this.channelDangerError = err.message;
          console.error(err);
          this.busy = false;
        });
    },
  },
  mounted: function () {
    this.findChannel();
  },
  beforeUnmount: function () {
    Timeouts.Abort("load-vod-info");
    Request.Abort("load-vod-info");
  },
  watch: {
    $route() {
      this.findChannel();
    }
  },
};
</script>