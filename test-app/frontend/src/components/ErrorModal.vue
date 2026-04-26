<template>
    <div class="modal-container" :class="{ hidden: !shown }" @click="close">
        <div class="modal">
            <div class="modal-message">{{ message }}</div>
            <div class="modal-controls">
                <button type="button" class="btn btn-primary btn-margin" @click="close">Ok</button>
            </div>
        </div>
    </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { useVModel } from "../utils/vmodel";

export default defineComponent({
    name: "ErrorModal",
    props: {
        shown: Boolean,
        message: String,
    },
    emits: ["update:shown"],
    setup(props) {
        return {
            shownState: useVModel(props, "shown"),
        };
    },
    data: function () {
        return {};
    },
    mounted: function () {},
    beforeUnmount: function () {},
    methods: {
        close: function () {
            this.shownState = false;
        },

        confirm: function (e: Event) {
            e.stopPropagation();
            this.shownState = false;
        },
    },
});
</script>
