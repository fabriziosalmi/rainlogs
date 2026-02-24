import DefaultTheme from 'vitepress/theme';
import { h } from 'vue';
import Rain from './Rain.vue';
import './custom.css';

export default {
    ...DefaultTheme,
    Layout() {
        return h(DefaultTheme.Layout, null, {
            'layout-top': () => h(Rain)
        });
    }
};
