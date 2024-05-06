// Control API

import { GetAPIURL, type RequestParams } from "@/utils/request";

export interface PublishingDetails {
    rtmp_base_url: string;
    wss_base_url: string;
}

export interface CreateChannelBody {
    id: string;
    record: boolean;
    resolutions: string;
    previews: string;
}

export interface UpdateChannelBody {
    key: string;
    record: boolean;
    resolutions: string;
    previews: string;
}

export interface ChannelChangedResponse {
    id: string;
    key: string;
    record: boolean;
    resolutions: string;
    previews: string;
}

export class ControlAPI {
    public static GetPublishingDetails(): RequestParams {
        return {
            method: "GET",
            url: GetAPIURL("/api/control"),
        };
    }

    public static CreateChannel(body: CreateChannelBody): RequestParams {
        return {
            method: "POST",
            url: GetAPIURL("/api/control/create"),
            json: body,
        };
    }

    public static UpdateChannel(channel: string, body: UpdateChannelBody): RequestParams {
        return {
            method: "POST",
            url: GetAPIURL("/api/control/chan/" + encodeURIComponent(channel) + ""),
            json: body,
        };
    }

    public static RefreshChannelKey(channel: string, key: string): RequestParams {
        return {
            method: "POST",
            url: GetAPIURL("/api/control/chan/" + encodeURIComponent(channel) + "/key"),
            json: {
                key: key,
            },
        };
    }

    public static CloseChannelStream(channel: string, key: string): RequestParams {
        return {
            method: "POST",
            url: GetAPIURL("/api/control/chan/" + encodeURIComponent(channel) + "/close"),
            json: {
                key: key,
            },
        };
    }

    public static DeleteVOD(channel: string, streamId: string, key: string): RequestParams {
        return {
            method: "POST",
            url: GetAPIURL("/api/control/chan/" + encodeURIComponent(channel) + "/vod/" + encodeURIComponent(streamId) + "/delete"),
            json: {
                key: key,
            },
        };
    }

    public static DeleteChannel(channel: string, key: string): RequestParams {
        return {
            method: "POST",
            url: GetAPIURL("/api/control/chan/" + encodeURIComponent(channel) + "/delete"),
            json: {
                key: key,
            },
        };
    }

    public static CheckKey(channel: string, key: string): RequestParams {
        return {
            method: "POST",
            url: GetAPIURL("/api/control/chan/" + encodeURIComponent(channel) + "/check"),
            json: {
                key: key,
            },
        };
    }
}
