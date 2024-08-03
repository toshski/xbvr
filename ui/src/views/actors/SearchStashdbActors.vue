<template>
  <b-modal :active="isModalActive"           
           has-modal-card
           trap-focus
           aria-role="dialog"
           @cancel="close"
           aria-modal
           can-cancel>
    

    <div class="modal-card" style="height: 80vh; width: 60vw; left: 10vw">
      <header class="modal-card-head">
        <p class="modal-card-title">Search Stashdb Actors</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>

      <div class="modal-card-body">
        <div >
          <b-field label="Find actor...">
            <b-input v-model="queryString" placeholder="Find actor..." @input="debouncedSearch" :loading="isFetching" custom-class="is-large"/>
          </b-field>
    
        <b-table :data="searchResults" >
          <b-table-column field="Name" >
            <template slot-scope="props">
              <div class="media">
                <div class="media-left">
                  <vue-load-image height="50px">
                    <img slot="image" :src="props.row.ImageUrl && props.row.ImageUrl.length ? getImageURL(props.row.ImageUrl[0]) : '/ui/images/blank_female_profile.png'" width="100" />
                    <img slot="preloader" src="/ui/images/blank.png" width="100" />
                    <img slot="error" src="/ui/images/blank.png" width="100" />
                  </vue-load-image>
                  <div v-if="props.row.DOB">
                    <small>
                      <strong>Birth Date:</strong> {{ format(parseISO(props.row.DOB), "yyyy-MM-dd") }}
                    </small>
                  </div>
                  <div>
                    <small>
                      <strong>Score:</strong> {{ props.row.Weight }}
                    </small>
                  </div>
                  <div>
                    <a class="button is-primary is-small" @click="linktoStashdb(props.row)" :title="'Link Actor with stashdb'">
                      <b-icon pack="mdi" :icon="'link-variant-plus'" size="is-small" />
                    </a>
                  </div>
                </div>
                <div class="media-content">
                  <div class="truncate">
                    <strong>
                      <a :href="props.row.Url" target="_blank">{{ props.row.Name }} - {{ props.row.Disambiguation }}</a>
                    </strong>
                  </div>
                </div>
              </div>
            </template>
          </b-table-column>
        </b-table>
      </div>
    </div>
    <footer class="modal-card-foot">
    </footer>
    </div>
  </b-modal>
</template>

<script>
import GlobalEvents from 'vue-global-events'
import ky from 'ky'
import VueLoadImage from 'vue-load-image'
import { format, parseISO } from 'date-fns'

function debounce(func, wait) {
  let timeout;
  return function(...args) {
    const context = this;
    clearTimeout(timeout);
    timeout = setTimeout(() => func.apply(context, args), wait);
  };
}

export default {
  name: 'SearchStashdbActors',
  components: {  GlobalEvents, VueLoadImage },
  data () {
    return {
        isModalActive: true,
        stashdbUrl: "",
        searchResults: [],
        queryString: "",
        isFetching: false,
        actor: "",
        }        
  },
  created() {
    this.debouncedSearch = debounce(this.searchStashdb, 750); // 750ms delay
  },
  mounted () {
    const item = Object.assign({}, this.$store.state.overlay.searchStashDbActors.actor)    
    this.actor = item
    this.openDialog(item)
    this.queryString=this.actor.name
  },
  methods: {
    format,
    parseISO,
    close () {
      this.$store.commit('overlay/hideSearchStashdbActors')
    },
    searchStashdb() {
        ky.get('/api/extref/stashdb/searchactor/' + this.actor.id + "?q=" + this.queryString, {timeout: 6e6}).json().then(data => {
            this.searchResults = Object.values(data.Results).sort((a, b) => b.Weight - a.Weight)
            this.isModalActive = true
            if (data.Status!='') {
              this.$buefy.toast.open({message: `Warning:  ${data.Status}`, type: 'is-warning', duration: 5000})
            }
        })
    },
    selectActor(option) {
        this.stashdbUrl=option.Url.replace("https://stashdb.org/performers/","")
        this.$nextTick(() => {
            if (this.$refs.autocompleteInput) {
                this.$refs.autocompleteInput.focus();
            }
        });
    },    
    linktoStashdb(option) {
        this.stashdbUrl=option.Url.replace("https://stashdb.org/performers/","")
        ky.get('/api/extref/stashdb/link2actor/' + this.actor.id +'/'+this.stashdbUrl ).json().then(data => {          
          // this.$store.commit('sceneList/updateScene', data)
           this.$store.commit('overlay/showActorDetails', { actor: data })
          this.close()
        })
    },    
    getImageURL (u) {        
      if (u != undefined && u.startsWith('http')) {
        return '/img/120x/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    openDialog(actor) {
        this.isModalActive = true
        this.searchStashdb()
        this.$nextTick(() => {
            if (this.$refs.autocompleteInput) {
                this.$refs.autocompleteInput.focus();
            }
        });
        this.actor = actor
    },
  },
  computed: {

}
}
</script>

<style scoped>
.b-modal {
  left: -20%;
  width: 40%;
  height: 65%;
  overflow: auto;
}

.tab-item {
  height: 40vh;
}
</style>
