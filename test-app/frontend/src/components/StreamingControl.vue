<template>
  <div class="main-content">
    <p>
      <RouterLink to="/">Back to channels list</RouterLink>
    </p>
    <div v-if="found">
      <h2>#{{ channelId }} -  <RouterLink :to="'/watch/' + channelId" target="_blank">/watch/{{ channelId }}</RouterLink></h2>
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
        <div class="form-group" v-if="live">
          <select class="form-control">
            <option :value="''">-- Select a quality to play it --</option>
            <option v-for="ss in liveSubStreams" :key="ss.indexFile" :value="ss.indexFile">{{ ss.width }}x{{ ss.height }}, {{ ss.fps }} fps</option>
          </select>
        </div>
        <div class="live-preview" v-if="live">
          <HLSPlayer url=""></HLSPlayer>
        </div>
      </div>

      <div class="channel-control-group">
        <h3>Configuration</h3>

        <div class="form-group">
          <label><input type="checkbox" v-model="hasOriginalResolution"> Enable encoding using the original resolution</label>
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
                  <td><button type="button" class="btn btn-danger">Delete</button></td>
                </tr>
                <tr>
                  <td><input type="number" placeholder="with (px)" class="form-control"/> x <input type="number" placeholder="height (px)" class="form-control"/></td>
                  <td><input type="number" placeholder="fps" class="form-control"/></td>
                  <td><button type="button" class="btn btn-primary">Add</button></td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="form-group">
          <label><input type="checkbox" v-model="record"> Enable recording (this will generate VODs of the streamings you publish)</label>
        </div>
        
        <div class="form-group" v-if="record">
          <label><input type="checkbox" v-model="previewsEnabled"> Enable preview images generation</label>
        </div>

        <div class="form-group" v-if="previewsEnabled">
          <label>Previews image width:</label>
          <input type="number" placeholder="with (px)" class="form-control" v-model="previewsWidth"/>
        </div>

        <div class="form-group" v-if="previewsEnabled">
          <label>Previews image height:</label>
          <input type="number" placeholder="height (px)" class="form-control" v-model="previewsHeight"/>
        </div>
        
        <div class="form-group" v-if="previewsEnabled">
          <label>Previews delay(seconds):</label>
          <input type="number" placeholder="delay (seconds)" class="form-control" v-model="previewsDelay"/>
        </div>

        <div class="form-group">
          <button type="button" class="btn btn-primary">Update channel configuration</button>
        </div>
      </div>

      <div class="channel-control-group">
        <h3>Danger zone</h3>
        <div class="form-group">
          <button type="button" class="btn btn-danger">Stop current active sessions</button>
        </div>
        <div class="form-group">
          <button type="button" class="btn btn-danger">Refresh streaming key</button>
        </div>
        <div class="form-group">
          <button type="button" class="btn btn-danger">Delete Channel</button>
        </div>
      </div>
    </div>
    <div v-if="!found">
      <h2>Channel not found</h2>
    </div>
  </div>
</template>
  
<script lang="ts">
import { ControlAPI, type PublishingDetails } from "@/api/api-control";
import { WatchAPI, type ChannelStatus, type SubStream } from "@/api/api-watch";
import { ChannelStorage } from "@/control/channel-storage";
import { parsePreviewsConfiguration } from "@/utils/previews-config";
import { Request } from "@/utils/request";
import { parseResolutionList, type Resolution } from "@/utils/resolutions";
import { Timeouts } from "@/utils/timeout";
import { defineComponent } from "vue";

import HLSPlayer from "./HLSPlayer.vue";
import { RouterLink } from 'vue-router';

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
}

export default defineComponent({
  name: "StreamingControl",
  emits: [],
  components: {
    HLSPlayer,
    RouterLink,
  },
  props: {
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

      rtmpBase: "",
      wssBase: "",
      loadingPublishingDetails: false,

      live: false,
      liveStartTimestamp: 0,
      liveSubStreams: [],
      liveNow: Date.now(),
    };
  },
  methods: {
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
  },
  mounted: function () {
    this.loadPublishingDetails();
    this.findChannel();
  },
  beforeUnmount: function () {
    Timeouts.Abort("load-publishing-details");
    Request.Abort("load-publishing-details");

    Timeouts.Abort("load-channel-status-control");
    Request.Abort("load-channel-status-control");
  },
  watch: {
    $route() {
      this.findChannel();
    }
  },
});
</script>