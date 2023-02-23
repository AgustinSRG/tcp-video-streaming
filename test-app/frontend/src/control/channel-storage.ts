/* Channel streaming storage */

"use strict";

const LOCAL_STORAGE_CONTROLLED_CHANNELS_KEY = "controlled_channels";

export interface ControlledChannel {
    id: string;
    key: string;

    record: boolean;
    resolutions: string;
    previews: string;
}

export class ChannelStorage {
    public static GetControlledChannels(): ControlledChannel[] {
        const storageContent = localStorage.getItem(LOCAL_STORAGE_CONTROLLED_CHANNELS_KEY) || "[]";

        try {
            let channels: ControlledChannel[] = JSON.parse(storageContent);

            if (!Array.isArray(channels)) {
                return [];
            }

            return channels;
        } catch (ex) {
            console.error(ex);
            return [];
        }
    }

    public static GetChannel(channel: string): ControlledChannel | null {
        const channels = ChannelStorage.GetControlledChannels();

        for (let ch of channels) {
            if (ch.id === channel) {
                return ch;
            }
        }

        return null;
    }

    public static SaveChannels(channels: ControlledChannel[]) {
        localStorage.setItem(LOCAL_STORAGE_CONTROLLED_CHANNELS_KEY, JSON.stringify(channels));
    }

    public static SetChannel(channel: ControlledChannel) {
        const channels = ChannelStorage.GetControlledChannels();

        let found = false;

        for (let ch of channels) {
            if (ch.id === channel.id) {
                ch.key = channel.key;
                ch.record = channel.record;
                ch.resolutions = channel.resolutions;
                ch.previews = channel.previews;
                found = true;
                break;
            }
        }

        if (!found) {
            channels.push({
                id: channel.id,
                key: channel.key,
                record: channel.record,
                resolutions: channel.resolutions,
                previews: channel.previews,
            });
        }

        ChannelStorage.SaveChannels(channels);
    }

    public static RemoveChannel(channel: string) {
        const channels = ChannelStorage.GetControlledChannels().filter(ch => {
            return channel !== ch.id;
        });

        ChannelStorage.SaveChannels(channels);
    }
}
