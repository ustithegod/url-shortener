import { createApp } from "https://unpkg.com/vue@3/dist/vue.esm-browser.prod.js";

createApp({
  data() {
    return {
      apiBase: "http://localhost:8082",
      form: {
        url: "",
        alias: "",
      },
      created: {},
      analyticsAlias: "",
      groupBy: "raw",
      analytics: null,
      createError: "",
      analyticsError: "",
      loading: {
        create: false,
        analytics: false,
      },
    };
  },
  methods: {
    async createShortUrl() {
      this.createError = "";
      this.created = {};
      this.loading.create = true;

      try {
        const response = await fetch(`${this.apiBase}/shorten`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            url: this.form.url,
            alias: this.form.alias || undefined,
          }),
        });

        const data = await response.json();
        if (!response.ok || data.error) {
          throw new Error(data.error || "Не удалось создать короткую ссылку");
        }

        this.created = data;
        this.analyticsAlias = data.alias;
      } catch (error) {
        this.createError = error.message;
      } finally {
        this.loading.create = false;
      }
    },
    async loadAnalytics() {
      this.analyticsError = "";
      this.analytics = null;
      this.loading.analytics = true;

      try {
        const response = await fetch(
          `${this.apiBase}/analytics/${encodeURIComponent(this.analyticsAlias)}?group_by=${encodeURIComponent(this.groupBy)}`,
        );
        const data = await response.json();

        if (!response.ok || data.error) {
          throw new Error(data.error || "Не удалось загрузить аналитику");
        }

        this.analytics = data;
      } catch (error) {
        this.analyticsError = error.message;
      } finally {
        this.loading.analytics = false;
      }
    },
    formatDate(value) {
      if (!value) {
        return "-";
      }

      return new Date(value).toLocaleString("ru-RU");
    },
  },
}).mount("#app");
