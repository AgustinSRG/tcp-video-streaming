// Watch API

import { GetAPIURL, type RequestParams } from "@/utils/request";

export interface SubStream {
    width: number;
    height: number;
    fps: number;
    indexFile: string;
}

export interface ChannelStatus {
    id: string;
    record: boolean;
    resolutions: string;
    previews: string;
    live: boolean;
    streamId: string;
    liveStartTimestamp: number;
    liveSubStreams: SubStream[];
}

export interface VODItem {
    streamId: string;
    timestamp: string;
}

export interface VODItemList {
    vod_list: VODItem[];
}

export interface VODStreaming {
    streamId: string;
    timestamp: number;
    subStreams: SubStream[];
    hasPreviews: boolean;
    previewsIndex: string;
}

export class WatchAPI {
    public static GetChannelStatus(channel: string): RequestParams {
        return {
            method: "GET",
            url: GetAPIURL("/api/watch/" + encodeURIComponent(channel)),
        };
    }

    public static GetChannelVODList(channel: string): RequestParams {
        return {
            method: "GET",
            url: GetAPIURL("/api/watch/" + encodeURIComponent(channel) + "/vod"),
        };
    }

    public static GetChannelVOD(channel: string, streamId: string): RequestParams {
        return {
            method: "GET",
            url: GetAPIURL("/api/watch/" + encodeURIComponent(channel) + "/vod/" + encodeURIComponent(streamId)),
        };
    }

    public static CreateAccount(username: string, password: string, write: boolean): RequestParams {
        return {
            method: "POST",
            url: GetAPIURL("/api/admin/accounts"),
            json: {
                username: username,
                password: password,
                write: write,
            },
        };
    }

    public static DeleteAccount(username: string): RequestParams {
        return {
            method: "POST",
            url: GetAPIURL("/api/admin/accounts/delete"),
            json: {
                username: username,
            },
        };
    }
}
