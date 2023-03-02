<template>
  <div class="image-previews-viewer">
    <div class="image-previews-controls">
      <span>Page: </span><input type="number" class="form-control" v-model.number="page" @input="updateImages"> of <b>{{
        pageCount }}</b> | <button class="btn btn-primary" @click="prevPage">Prev</button> | <button
        class="btn btn-primary" @click="nextPage">Next</button> | Page size: <input type="number" class="form-control"
        v-model.number="pageSize" @input="updateImages">
    </div>
    <div class="image-previews-list">
      <div class="img-preview" v-for="img in images" :key="img.name">
        <img :src="img.url" :title="img.name" :alt="img.name">
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { Request } from '@/utils/request';
import { Timeouts } from '@/utils/timeout';

interface ImagePreviewsIndex {
  index_start: number;
  count: number;
  pattern: string;
}

interface ComponentData {
  index_start: number;
  count: number;
  pattern: string;

  images: {
    url: string;
    name: string;
  }[];

  page: number;
  pageCount: number;
  pageSize: number;
}

export default {
  name: "ImagePreviewsViewer",
  props: {
    url: String,
  },
  data: function (): ComponentData {
    return {
      index_start: 0,
      count: 0,
      pattern: "%d.jpg",

      images: [],
      page: 0,
      pageCount: 1,
      pageSize: 20,
    };
  },
  methods: {
    load: function () {
      if (!this.url) {
        return;
      }

      Timeouts.Abort("load-image-viewer");

      Request.Pending("load-image-viewer",
        { method: "GET", url: this.url }
      )
        .onSuccess((result: ImagePreviewsIndex) => {
          this.index_start = result.index_start;
          this.count = result.count;
          this.pattern = result.pattern;
          this.page = 0;
          this.updateImages();
        })
        .onRequestError((err) => {
          Request.ErrorHandler()
            .add("*", "*", () => {
              Timeouts.Set("load-image-viewer", 2000, this.load.bind(this));
            })
            .handle(err);
        })
        .onUnexpectedError((err) => {
          console.error(err);
          Timeouts.Set("load-image-viewer", 2000, this.load.bind(this));
        });
    },

    nextPage: function () {
      this.page++;
      this.updateImages();
    },

    prevPage: function () {
      this.page--;
      this.updateImages();
    },

    updateImages: function () {
      this.pageSize = Math.floor(this.pageSize);

      if (this.pageSize <= 0) {
        this.pageSize = 1;
      }

      let pageCount = Math.floor(this.count / this.pageSize);

      if (this.count % this.pageSize !== 0) {
        pageCount++;
      }

      if (pageCount <= 0) {
        pageCount = 1;
      }

      this.pageCount = pageCount;

      this.page = Math.floor(this.page);

      if (this.page < 1) {
        this.page = 1;
      }

      if (this.page > pageCount) {
        this.page = pageCount;
      }

      const images: {
        url: string;
        name: string;
      }[] = [];

      const urlParts = (this.url + "").split("/");
      const folder = urlParts.slice(0, urlParts.length - 1).join("/");

      const pageStart = (this.page - 1) * this.pageSize;
      const pageEnd = Math.min(pageStart + this.pageSize, this.count);

      for (let i = pageStart; i < pageEnd; i++) {
        const trueIndex = this.index_start + i;
        const name = this.pattern.replace("%d", "" + trueIndex);
        images.push({
          name: name,
          url: folder + "/" + name,
        });
      }

      this.images = images;
    },
  },
  mounted: function () {
    this.load();
  },
  beforeUnmount: function () {

  },
  watch: {
    url: function () {
      this.load();
    },
  }
};
</script>

<style>
.image-previews-list {
  display: flex;
  flex-wrap: wrap;
  padding-top: 0.5rem;
}

.img-preview {
  margin: 0.2rem;
}
</style>
