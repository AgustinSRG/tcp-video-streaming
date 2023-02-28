<template>
  <div class="main-content">
    <p>
      <RouterLink to="/">Back to channels list</RouterLink>
    </p>
    <div v-if="found">
      <h2>#{{ channelId }} - <RouterLink :to="'/watch/' + channelId" target="_blank">/watch/{{ channelId }}</RouterLink>
      </h2>
      <div class="channel-control-group">
        <h3>RTMP Publishing Details</h3>
        <p v-if="loadingPublishingDetails">Loading...</p>
        <div class="form-group" v-if="!loadingPublishingDetails">
          <label>Streaming key:</label>
          <details>
            <summary>Click to reveal</summary>
            <b>{{ channelKey }}</b>
          </details>
        </div>
        <div class="form-group" v-if="!loadingPublishingDetails">
          <label>RTMP URL for publishing:</label>
          <details>
            <summary>Click to reveal</summary>
            <b>{{ (rtmpBase + "/" + channelId + "/" + channelKey) }}</b>
          </details>
        </div>
      </div>

      <div class="channel-control-group">
        <h3>Publish from your browser</h3>
        <p v-if="loadingPublishingDetails">Loading...</p>
        <div class="form-group" v-if="!loadingPublishingDetails">
          <button type="button" class="btn btn-primary btn-margin">Publish from camera</button>
          <button type="button" class="btn btn-primary btn-margin">Publish screen share</button>
          <button type="button" class="btn btn-primary btn-margin">Stop publishing</button>
        </div>
      </div>

      <div class="channel-control-group">
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
        <div class="live-preview" v-if="live">
          <HLSPlayer :url="getHLSURL(selectedLiveSubStream, liveSubStreams)"></HLSPlayer>
        </div>
      </div>

      <div class="channel-control-group">
        <h3>Configuration</h3>

        <div class="form-group">
          <label><input type="checkbox" v-model="hasOriginalResolution"> Enable encoding using the original
            resolution</label>
        </div>

        <div class="form-group">
          <label>Extra resolutions:</label>
          <div class="table-container">
            <table class="table">
              <thead>
                <tr>
                  <th>Resolution</th>
                  <th>Frames per second</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="r in resolutions" :key="r.width + 'x' + r.height + '-' + r.fps">
                  <td>{{ r.width }}x{{ r.height }}</td>
                  <td>{{ r.fps }}</td>
                  <td><button type="button" class="btn btn-danger" :disabled="busy"
                      @click="deleteResolution(r)">Delete</button></td>
                </tr>
                <tr>
                  <td><input type="number" v-model.number="resToAddWidth" :disabled="busy" placeholder="with (px)"
                      class="form-control" /> x <input type="number" v-model.number="resToAddHeight" :disabled="busy"
                      placeholder="height (px)" class="form-control" /></td>
                  <td><input type="number" v-model.number="resToAddFps" :disabled="busy" placeholder="fps"
                      class="form-control" /></td>
                  <td><button type="button" class="btn btn-primary" :disabled="busy" @click="addResolution">Add</button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="form-group">
          <label><input type="checkbox" v-model="record" :disabled="busy"> Enable recording (this will generate VODs of
            the streamings you publish)</label>
        </div>

        <div class="form-group" v-if="record">
          <label><input type="checkbox" v-model="previewsEnabled" :disabled="busy"> Enable preview images
            generation</label>
        </div>

        <div class="form-group" v-if="previewsEnabled">
          <label>Previews image width:</label>
          <input type="number" placeholder="with (px)" class="form-control" :disabled="busy"
            v-model.number="previewsWidth" />
        </div>

        <div class="form-group" v-if="previewsEnabled">
          <label>Previews image height:</label>
          <input type="number" placeholder="height (px)" class="form-control" :disabled="busy"
            v-model.number="previewsHeight" />
        </div>

        <div class="form-group" v-if="previewsEnabled">
          <label>Previews delay(seconds):</label>
          <input type="number" placeholder="delay (seconds)" class="form-control" :disabled="busy"
            v-model.number="previewsDelay" />
        </div>

        <div class="form-group">
          <div class="form-error" v-if="channelConfigError">{{ channelConfigError }}</div>
          <button type="button" class="btn btn-primary" @click="updateConfiguration" :disabled="busy">Update channel
            configuration</button>
        </div>
      </div>

      <div class="channel-control-group">
        <h3>Danger zone</h3>
        <div class="form-error" v-if="channelDangerError">{{ channelDangerError }}</div>
        <div class="form-group">
          <button type="button" class="btn btn-danger" :disabled="!live || busy" @click="askStop">Stop current active
            sessions</button>
        </div>
        <div class="form-group">
          <button type="button" class="btn btn-danger" :disabled="busy" @click="askRefresh">Refresh streaming key</button>
        </div>
        <div class="form-group">
          <button type="button" class="btn btn-danger" :disabled="busy" @click="askDelete">Delete Channel</button>
        </div>
      </div>
    </div>
    <div v-if="!found">
      <h2>Channel not found</h2>
    </div>

    <ConfirmationModal v-model:shown="displayConfirmDelete" message="Delete this channel? (This action is not reversible)"
      @confirm="doDelete"></ConfirmationModal>
    <ConfirmationModal v-model:shown="displayConfirmStop" message="Stop active streams?" @confirm="doStop">
    </ConfirmationModal>
    <ConfirmationModal v-model:shown="displayConfirmRefresh"
      message="Refresh streaming key? (The old one will be invalidated)" @confirm="doRefresh"></ConfirmationModal>
  </div>
</template>
  
<script lang="ts">
import { ControlAPI, type ChannelChangedResponse, type PublishingDetails } from "@/api/api-control";
import { WatchAPI, type ChannelStatus, type SubStream } from "@/api/api-watch";
import { ChannelStorage } from "@/control/channel-storage";
import { encodePreviewsConfiguration, parsePreviewsConfiguration } from "@/utils/previews-config";
import { GetAssetURL, Request } from "@/utils/request";
import { encodeResolutionList, parseResolutionList, type Resolution } from "@/utils/resolutions";
import { Timeouts } from "@/utils/timeout";

import HLSPlayer from "./HLSPlayer.vue";
import ConfirmationModal from "./ConfirmationModal.vue";
import { RouterLink } from 'vue-router';
import router from "@/router";
import { renderTimeSeconds } from "@/utils/time-utils";

interface ComponentData {
  found: boolean;
  channelId: string;
  channelKey: string;
  record: boolean;
  hasOriginalResolution: boolean;
  resolutions: Resolution[];
  previewsEnabled: boolean;
  previewsWidth: number;
  previewsHeight: number;
  previewsDelay: number;

  rtmpBase: string;
  wssBase: string;
  loadingPublishingDetails: boolean;

  live: boolean;
  liveStartTimestamp: number;
  liveNow: number;
  liveSubStreams: SubStream[];
  selectedLiveSubStream: string,

  displayConfirmDelete: boolean;
  displayConfirmStop: boolean;
  displayConfirmRefresh: boolean;

  resToAddWidth: number;
  resToAddHeight: number;
  resToAddFps: number;

  busy: boolean;

  channelConfigError: string;
  channelDangerError: string;
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
      record: false,
      hasOriginalResolution: false,
      resolutions: [],
      previewsEnabled: false,
      previewsWidth: 0,
      previewsHeight: 0,
      previewsDelay: 0,

      resToAddWidth: 1024,
      resToAddHeight: 720,
      resToAddFps: 30,

      rtmpBase: "",
      wssBase: "",
      loadingPublishingDetails: false,

      live: false,
      liveStartTimestamp: 0,
      liveSubStreams: [],
      liveNow: Date.now(),
      selectedLiveSubStream: "",

      displayConfirmDelete: false,
      displayConfirmStop: false,
      displayConfirmRefresh: false,

      busy: false,

      channelConfigError: "",
      channelDangerError: "",
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

    renderTime: function (t: number) {
      return renderTimeSeconds(Math.round(t / 1000))
    },

    findChannel: function () {
      const channel = ChannelStorage.GetChannel(this.$route.params.channel + "");

      if (!channel) {
        this.found = false;
        return;
      }

      this.found = true;
      this.channelId = channel.id;
      this.channelKey = channel.key;
      this.record = channel.record;

      const resList = parseResolutionList(channel.resolutions);

      this.hasOriginalResolution = resList.hasOriginal;
      this.resolutions = resList.resolutions;

      const previewConfig = parsePreviewsConfiguration(channel.previews);

      this.previewsEnabled = previewConfig.enabled;
      this.previewsWidth = previewConfig.width;
      this.previewsHeight = previewConfig.height;
      this.previewsDelay = previewConfig.delaySeconds;

      this.live = false;
      this.loadChannelStatus();
    },

    loadPublishingDetails: function () {
      this.loadingPublishingDetails = true;

      Timeouts.Abort("load-publishing-details");

      Request.Pending("load-publishing-details",
        ControlAPI.GetPublishingDetails()
      )
        .onSuccess((result: PublishingDetails) => {
          this.rtmpBase = result.rtmp_base_url;
          this.wssBase = result.wss_base_url;
          this.loadingPublishingDetails = false;
        })
        .onRequestError((err) => {
          Request.ErrorHandler()
            .add("*", "*", () => {
              Timeouts.Set("load-publishing-details", 2000, this.loadPublishingDetails.bind(this));
            })
            .handle(err);
        })
        .onUnexpectedError((err) => {
          console.error(err);
          Timeouts.Set("load-publishing-details", 2000, this.loadPublishingDetails.bind(this));
        });
    },

    loadChannelStatus: function () {
      Timeouts.Abort("load-channel-status-control");

      Request.Pending("load-channel-status-control",
        WatchAPI.GetChannelStatus(this.channelId)
      )
        .onSuccess((result: ChannelStatus) => {
          this.live = result.live;
          this.liveStartTimestamp = result.liveStartTimestamp;
          this.liveSubStreams = result.liveSubStreams;
          Timeouts.Set("load-channel-status-control", 2000, this.loadChannelStatus.bind(this));
        })
        .onRequestError((err) => {
          Request.ErrorHandler()
            .add("*", "*", () => {
              Timeouts.Set("load-channel-status-control", 2000, this.loadChannelStatus.bind(this));
            })
            .handle(err);
        })
        .onUnexpectedError((err) => {
          console.error(err);
          Timeouts.Set("load-channel-status-control", 2000, this.loadChannelStatus.bind(this));
        });
    },

    addResolution: function () {
      const width = this.resToAddWidth;
      const height = this.resToAddHeight;
      const fps = this.resToAddFps;
      const key = width + "x" + height + "-" + fps;

      for (let res of this.resolutions) {
        const otherKey = res.width + "x" + res.height + "-" + res.fps;
        if (key === otherKey) {
          return;
        }
      }

      this.resolutions.push({
        width: width,
        height: height,
        fps: fps,
      });
    },

    deleteResolution: function (res: Resolution) {
      const key = res.width + "x" + res.height + "-" + res.fps;

      this.resolutions = this.resolutions.filter(r => {
        const otherKey = r.width + "x" + r.height + "-" + r.fps;
        return key !== otherKey;
      });
    },

    updateConfiguration: function () {
      if (this.busy) {
        return;
      }

      this.busy = true;
      this.channelConfigError = "";

      Request.Do(
        ControlAPI.UpdateChannel(this.channelId, {
          key: this.channelKey,
          record: this.record,
          resolutions: encodeResolutionList({
            hasOriginal: this.hasOriginalResolution,
            resolutions: this.resolutions,
          }),
          previews: encodePreviewsConfiguration({
            enabled: this.record && this.previewsEnabled,
            width: this.previewsWidth,
            height: this.previewsHeight,
            delaySeconds: this.previewsDelay,
          }),
        })
      )
        .onSuccess((result: ChannelChangedResponse) => {
          this.busy = false;
          ChannelStorage.SetChannel(result);
          this.findChannel();
        })
        .onCancel(() => {
          this.busy = false;
        })
        .onRequestError((err) => {
          this.busy = false;
          Request.ErrorHandler()
            .add(400, "INVALID_RESOLUTIONS", () => {
              this.channelConfigError = "The resolutions are invalid. Check they have valid dimensions.";
            })
            .add(400, "INVALID_PREVIEWS_CONFIG", () => {
              this.channelConfigError = "The previews configuration is invalid.";
            })
            .add(400, "*", () => {
              this.channelConfigError = "Bad request";
            })
            .add(403, "*", () => {
              this.channelConfigError = "Access denied. This may indicate the channel was deleted or the key changed.";
            })
            .add(404, "*", () => {
              this.channelConfigError = "Channel not found. This may indicate the channel was deleted.";
            })
            .add(500, "*", () => {
              this.channelConfigError = "Internal server error";
            })
            .add("*", "*", () => {
              this.channelConfigError = "Could not connect to the server";
            })
            .handle(err);
        })
        .onUnexpectedError((err) => {
          this.channelConfigError = err.message;
          console.error(err);
          this.busy = false;
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
        ControlAPI.DeleteChannel(this.channelId, this.channelKey)
      )
        .onSuccess(() => {
          this.busy = false;
          ChannelStorage.RemoveChannel(this.channelId);
          router.push('/');
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
              ChannelStorage.RemoveChannel(this.channelId);
              router.push('/');
            })
            .add(404, "*", () => {
              ChannelStorage.RemoveChannel(this.channelId);
              router.push('/');
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

    askStop: function () {
      this.displayConfirmStop = true;
    },

    doStop: function () {
      if (this.busy) {
        return;
      }

      this.busy = true;
      this.channelDangerError = "";

      Request.Do(
        ControlAPI.CloseChannelStream(this.channelId, this.channelKey)
      )
        .onSuccess(() => {
          this.busy = false;
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
              this.channelDangerError = "Access denied. This may indicate the channel was deleted or the key changed.";
            })
            .add(404, "*", () => {
              this.channelDangerError = "Channel not found. This may indicate the channel was deleted.";
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

    askRefresh: function () {
      this.displayConfirmRefresh = true;
    },

    doRefresh: function () {
      if (this.busy) {
        return;
      }

      this.busy = true;
      this.channelDangerError = "";

      Request.Do(
        ControlAPI.RefreshChannelKey(this.channelId, this.channelKey)
      )
        .onSuccess((result: ChannelChangedResponse) => {
          this.busy = false;
          ChannelStorage.SetChannel(result);
          this.findChannel();
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
              this.channelDangerError = "Access denied. This may indicate the channel was deleted or the key changed.";
            })
            .add(404, "*", () => {
              this.channelDangerError = "Channel not found. This may indicate the channel was deleted.";
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
    this.loadPublishingDetails();
    this.findChannel();

    this.$options.tickTimer = setInterval(this.updateNow.bind(this), 500);
  },
  beforeUnmount: function () {
    Timeouts.Abort("load-publishing-details");
    Request.Abort("load-publishing-details");

    Timeouts.Abort("load-channel-status-control");
    Request.Abort("load-channel-status-control");

    clearInterval(this.$options.tickTimer);
  },
  watch: {
    $route() {
      this.findChannel();
    }
  },
};
</script>