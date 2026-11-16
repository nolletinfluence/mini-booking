import '@fontsource-variable/roboto'
import 'primeicons/primeicons.css'
import PrimeVue from 'primevue/config'
import { createApp } from 'vue'
import SchedulePage from '@/pages/schedule'
import './styles.css'

createApp(SchedulePage)
  .use(PrimeVue, { unstyled: true, locale: { aria: { close: 'Закрыть' } } })
  .mount('#app')
