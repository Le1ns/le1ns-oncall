import { AppPlugin } from '@grafana/data';
import { App } from './components/App';
import './styles.css';

export const plugin = new AppPlugin().setRootPage(App);
