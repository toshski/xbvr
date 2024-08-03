<!--
   accept query string back  

   add a rescan option for users entering their own query, try to validate
-->
<template>
  <b-modal :active="isModalActive"           
           has-modal-card
           trap-focus
           aria-role="dialog"
           @cancel="close"
           aria-modal
           can-cancel>
    

    <div class="modal-card" id="test" style="height: 65vh; width: 40vw; left: -20vw">
      <header class="modal-card-head">
        <p class="modal-card-title">Search Stashdb</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>

      <div class="modal-card-body">
            <div >
                <!-- <div><span class="has-text-danger is-small">warnindgS</span></div>                 -->
                <b-field>                    
                    <b-autocomplete
                        ref="autocompleteInput"
                        :data="searchResults"
                        placeholder="Find scene..."
                        field="query"
                        :loading="isFetching"
                        v-model="queryString"
                        @typing="searchStashdb"
                        @select="option => selectScene(option)"
                        :open-on-focus="true"
                        custom-class="is-large"
                        max-height="450">

                        <template slot-scope="props" >
                            <div class="media">
                                <div class="media-left">
                                    <vue-load-image>
                                        <img slot="image" :src="getImageURL(props.option.ImageUrl)" height="150"/>
                                        <img slot="preloader" src="/ui/images/blank.png" height="150"/>
                                        <img slot="error" src="/ui/images/blank.png" height="150"/>
                                    </vue-load-image>
                                  <div v-if="props.option.Date!=''"><small><strong>Released:</strong> {{format(parseISO(props.option.Date), "yyyy-MM-dd")}}</small></div>
                                  <div v-if="props.option.Duration!=''"><small><strong>Durn:</strong> {{ props.option.Duration }}</small></div>
                                  <div><small><strong>Score:</strong> {{ props.option.Weight }}</small></div>
                                  <div>
                                    <a class="button is-primary is-small" @click="linktoStashdb(props.option)" :title="'Link scene with stashdb'">
                                      <b-icon pack="mdi" :icon="'link-variant-plus'" size="is-small"/>
                                    </a>
                                  </div>
                                </div>
                                <div class="media-content">
                                    <div class="truncate"><strong><a :href="props.option.Url"  target="_blank">{{ props.option.Studio }} - {{ props.option.Title }}</a></strong></div>                                    
                                    <div><small style="white-space: normal; display: block;">{{props.option.Description}}</small></div>
                                    <div style="margin-top:0.5em">                                        
                                        <small style="white-space: normal; display: block;">
                                            <span v-for="(c, idx) in props.option.Performers" :key="'Performers' + idx">
                                                {{c.Name}}<span v-if="idx < props.option.Performers.length-1">, </span>
                                            </span>
                                        </small>
                                    </div>
                                </div>            
                            </div>
                        </template>
                    </b-autocomplete>
                </b-field>

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

export default {
  name: 'SearchStashdb',
  components: {  GlobalEvents, VueLoadImage },
  data () {
    return {
        isModalActive: true,
        stashdbUrl: "",
        searchResults: [],
        query: "",
        queryString: "",
        isFetching: false,
        scene: "",
        }        
  },
  mounted () {
    const item = Object.assign({}, this.$store.state.overlay.searchStashDb.scene)    
    console.log("insearch stash")
    console.log(item)
    this.scene = item
    this.openDialog(item)    
  },
  methods: {
    format,
    parseISO,
    close () {
      console.log("close")
      this.$store.commit('overlay/hideSearchStashdb')
    },
    searchStashdb() {
        console.log("in searchStashdb", this.queryString)        
        console.log("scene ", this.scene)        
        ky.get('/api/extref/stashdb/search/' + this.scene.id + "?q=" + this.queryString, {timeout: 6e6}).json().then(data => {
            this.searchResults = Object.values(data.Results).sort((a, b) => b.Weight - a.Weight)
            console.log("search results")
            console.log(this.searchResults)
            console.log("search results isModalActive", this.isModalActive)
            this.isModalActive = true
            if (data.Status!='') {
              this.$buefy.toast.open({message: `Warning:  ${data.Status}`, type: 'is-warning', duration: 5000})
            }
        })
    },
    selectScene(option) {
        console.log("in select scene")
        console.log(option)
        this.stashdbUrl=option.Url.replace("https://stashdb.org/scenes/","")
        this.$nextTick(() => {
          console.log("nextTick", this.isModalActive)
            if (this.$refs.autocompleteInput) {
                this.$refs.autocompleteInput.focus();
            }
        });
    },    
    linktoStashdb(option) {
        console.log("in linktoStashdb")
        console.log(option)
        this.stashdbUrl=option.Url.replace("https://stashdb.org/scenes/","")
        console.log('/api/extref/stashdb/link2scene/' + this.scene.id +'/'+this.stashdbUrl)
        ky.get('/api/extref/stashdb/link2scene/' + this.scene.id +'/'+this.stashdbUrl ).json().then(data => {          
          this.$store.commit('sceneList/updateScene', data)
          this.$store.commit('overlay/showDetails', { scene: data })
          this.close()
        })
        //ky.get('/api/extref/stashdb/search/' + this.item.id, {searchParams: { stashid: url },timeout: 60000}).json().then(data => {
            //ky.get('/api/extref/stashdb/search/' + this.item.id, {timeout: 6e6}).json().then(data => {        
    },    
    getImageURL (u) {        
      if (u != undefined && u.startsWith('http')) {
        return '/img/120x/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    openDialog(scene) {
      console.log("in openDialogOld")
        this.isModalActive = true
        console.log("openDialogOld isModalActive", this.isModalActive)
        this.searchStashdb()
        console.log("openDialogOld2 isModalActive", this.isModalActive)
        this.$store.commit('overlay/changeDetailsTab', { tab: 3 })
        this.$nextTick(() => {
          console.log("nextTick", this.isModalActive)
            if (this.$refs.autocompleteInput) {
                this.$refs.autocompleteInput.focus();
            }
        });
        this.scene = scene
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
