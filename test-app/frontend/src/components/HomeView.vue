<template>
  <div class="main-content">
    <form @submit="createChannelSubmit">
      <div class="form-group">
        <label>Channel ID (Can only contain letters, numbers, underscores and hyphens): </label>
        <input type="text" class="form-control" v-model="channelIdToCreate" placeholder="my_streaming_channel"
          :disabled="busy">
      </div>
      <div class="form-group">
        <div class="form-error" v-if="channelCreateError">{{ channelCreateError }}</div>
        <button class="btn btn-primary" type="submit" :disabled="!channelIdToCreate">Create channel</button>
      </div>

    </form>

    <hr />

    <div class="table-container">
      <p v-if="channels.length === 0">You have no streaming channels under your control.</p>
      <table v-if="channels.length > 0" class="table table-full">
        <thead>
          <tr>
            <th>Channel</th>
            <th>Watch URL</th>
            <th>Recording</th>
            <th>HLS Resolutions</th>
            <th>Preview images</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="ch in channels" :key="ch.id">
            <td>
              <RouterLink :to="'/control/' + ch.id">#{{ ch.id }}</RouterLink>
            </td>
            <td>
              <RouterLink :to="'/watch/' + ch.id" target="_blank">/watch/{{ ch.id }}</RouterLink>
            </td>
            <td>{{ ch.record ? 'Enabled' : 'Disabled' }}</td>
            <td>{{ ch.resolutions }}</td>
            <td>{{ ch.previews }}</td>
            
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
  
<script lang="ts">
import { ControlAPI, type ChannelChangedResponse } from '@/api/api-control';
import { ChannelStorage, type ControlledChannel } from '@/control/channel-storage';
import { Request } from '@/utils/request';
import { RouterLink } from 'vue-router'

interface ComponentData {
  channels: ControlledChannel[];
  channelIdToCreate: string;
  channelCreateError: string;
  busy: boolean;
}

export default {
  name: "HomeView",
  components: {
    RouterLink
  },
  emits: [],
  data: function (): ComponentData {
    return {
      channels: [],
      channelIdToCreate: "",
      channelCreateError: "",
      busy: false,
    };
  },
  methods: {
    createChannelSubmit: function (event: Event) {
      event.preventDefault();

      if (this.busy) {
        return;
      }

      this.busy = true;

      Request.Do(
        ControlAPI.CreateChannel({ id: this.channelIdToCreate, record: false, resolutions: "", previews: "" })
      )
        .onSuccess((result: ChannelChangedResponse) => {
          this.busy = false;
          this.channelIdToCreate = "";
          ChannelStorage.SetChannel(result);
          this.loadChannels();
        })
        .onCancel(() => {
          this.busy = false;
        })
        .onRequestError((err) => {
          this.busy = false;
          Request.ErrorHandler()
            .add(400, "INVALID_CHANNEL_ID", () => {
              this.channelCreateError = "Invalid channel ID";
            })
            .add(400, "ID_TAKEN", () => {
              this.channelCreateError = "There is already another channel with that identifier. Please, choose another one.";
            })
            .add(400, "*", () => {
              this.channelCreateError = "Bad request";
            })
            .add(500, "*", () => {
              this.channelCreateError = "Internal server error";
            })
            .add("*", "*", () => {
              this.channelCreateError = "Could not connect to the server";
            })
            .handle(err);
        })
        .onUnexpectedError((err) => {
          this.channelCreateError = err.message;
          console.error(err);
          this.busy = false;
        });
    },

    loadChannels: function () {
      this.channels = ChannelStorage.GetControlledChannels();
    },
  },
  mounted: function () {
    this.loadChannels();
  },
  beforeUnmount: function () { },
};
</script>