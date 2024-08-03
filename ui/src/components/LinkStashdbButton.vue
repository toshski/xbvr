<template>
    <span>
        <a :class="buttonClass"
            @click="openDialog()"
            :title="'Link to stashdb'">
            <b-icon pack="mdi" :icon="'link-variant-plus'" size="is-small"/>
        </a>
        <b-modal :active.sync="isModalActive"
                has-modal-card
                trap-focus
                aria-role="dialog"
                aria-modal>
            <div class="modal-card" style="width: 1000">
            <header class="modal-card-head">
                <p class="modal-card-title">{{$t('Enter Stashdb Link')}}</p>
            </header>
            <section class="modal-card-body">
                <b-field label="Stashdb url">
                    <b-input v-model='stashdbUrl' />
                </b-field>
            </section>
            <footer class="modal-card-foot">
                <button class="button is-primary" :disabled="this.stashdbUrl == ''" @click="linktoStashdb()">Link</button>
            </footer>
            </div>
        </b-modal>
    </span>
</template>

<script>
import ky from 'ky'
export default {
  name: 'LinkStashdbButton',
  props: { item: Object },
  data () {
    return {
        isModalActive: false,
        stashdbUrl: ""
        }
  },
  computed: {
    buttonClass () {
      return 'button  is-outlined is-small'
    }
  },
  methods: {
    openDialog() {
        this.isModalActive = true
    },
    linktoStashdb() {
        this.isModalActive = false
        this.stashdbUrl=this.stashdbUrl.replace("https://stashdb.org/scenes/","")
        ky.get('/api/extref/stashdb/link2scene/' + this.item.id +'/'+this.stashdbUrl )
    },
  }
}
</script>
