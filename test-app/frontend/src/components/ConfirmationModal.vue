<template>
  <div class="modal-container" :class="{hidden: !shown}" @click="close">
    <div class="modal">
      <div class="modal-message">{{ message }}</div>
      <div class="modal-controls">
        <button type="button" class="btn btn-primary btn-margin" @click="confirm">Yes</button>
        <button type="button" class="btn btn-primary btn-margin" @click="close">No</button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { useVModel } from "../utils/vmodel";

export default defineComponent({
  name: "ConfirmationModal",
  emits: ["confirm", "update:shown"],
  props: {
    shown: Boolean,
    message: String,
  },
  setup(props) {
    return {
      shownState: useVModel(props, "shown"),
    };
  },
  data: function () {
    return {
    };
  },
  methods: {
    close: function () {
      this.shownState = false;
    },

    confirm: function (e: Event) {
      e.stopPropagation();
      this.$emit("confirm");
      this.shownState = false;
    },
  },
  mounted: function () {},
  beforeUnmount: function () {},
});
</script>

