// Custom v-model for composition API

import { computed, getCurrentInstance, type ComponentInternalInstance } from 'vue'

export const useVModel = (props: any, propName: string) => {
    const vm: any = (<ComponentInternalInstance>getCurrentInstance()).proxy

    return computed({
        get() {
            return props[propName]
        },
        set(value) {
            vm.$emit(`update:${propName}`, value)
        },
    })
}
