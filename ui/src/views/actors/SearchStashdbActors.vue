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
    

    <div class="modal-card" id="test" style="height: 80vh; width: 60vw; left: 10vw">
      <header class="modal-card-head">
        <p class="modal-card-title">Search Stashdb Actors</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>

      <div class="modal-card-body">
            <div >
                <!-- <div><span class="has-text-danger is-small">warnindgS</span></div>                 -->
                  <b-field label="Find actor...">
      <b-input
        v-model="queryString"
        placeholder="Find actor..."
        @input="debouncedSearch"
        :loading="isFetching"
        custom-class="is-large"
      ></b-input>
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
                    <b-field v-if="false">                                      
                    <b-autocomplete
                        ref="autocompleteInput"
                        :data="searchResults"
                        placeholder="Find actor..."
                        field="query"
                        :loading="isFetching"
                        v-model="queryString"
                        @typing="searchStashdb"
                        @select="option => selectActor(option)"
                        :open-on-focus="true"
                        custom-class="is-large"
                        max-height="70vh">

                        <template slot-scope="props" >
                            <div class="media">
                                <div class="media-left">
                                    <vue-load-image>
                                        <img slot="image" :src="props.option.ImageUrl!=null && props.option.ImageUrl.length !=0 ? getImageURL(props.option.ImageUrl[0]) : '/ui/images/blank_female_profile.png'" height="150" width="200"/>
                                        <img slot="preloader" src="/ui/images/blank.png" height="150" width="200"/>
                                        <img slot="error" src="/ui/images/blank.png" height="150"  width="200"/>
                                    </vue-load-image>
                                  <div v-if="props.option.DOB!=''"><small><strong>Birth Date:</strong> {{format(parseISO(props.option.DOB), "yyyy-MM-dd")}}</small></div>                                  
                                  <div><small><strong>Score:</strong> {{ props.option.Weight }}</small></div>
                                  <div>
                                    <a class="button is-primary is-small" @click="linktoStashdb(props.option)" :title="'Link Actor with stashdb'">
                                      <b-icon pack="mdi" :icon="'link-variant-plus'" size="is-small"/>
                                    </a>
                                  </div>
                                </div>
                                <div class="media-content">
                                     <div class="truncate"><strong><a :href="props.option.Url"  target="_blank">{{ props.option.Name }} - {{ props.option.Disambiguation }}</a></strong></div>                                                                        
<!--                                    <div style="margin-top:0.5em">                                        
                                        <small style="white-space: normal; display: block;">
                                            <span v-for="(c, idx) in props.option.Performers" :key="'Performers' + idx">
                                                {{c.Name}}<span v-if="idx < props.option.Performers.length-1">, </span>
                                            </span>
                                        </small>
                                    </div>
 -->                            </div>
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
        query: "",
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
    console.log("insearch stash")
    console.log(item)
    this.actor = item
    this.openDialog(item)
    this.queryString=this.actor.name
    this.query=this.actor.name
  },
  methods: {
    format,
    parseISO,
    close () {
      console.log("close")
      this.$store.commit('overlay/hideSearchStashdbActors')
    },
    searchStashdb() {
        console.log("in searchStashdb", this.queryString)        
        console.log("in searchStashdb this ", this)        
        console.log("actor ", this.actor)        
        ky.get('/api/extref/stashdb/searchactor/' + this.actor.id + "?q=" + this.queryString, {timeout: 6e6}).json().then(data => {
            this.searchResults = Object.values(data.Results).sort((a, b) => b.Weight - a.Weight)
            console.log(this.searchResults)
            this.isModalActive = true
            if (data.Status!='') {
              this.$buefy.toast.open({message: `Warning:  ${data.Status}`, type: 'is-warning', duration: 5000})
            }
        })
    },
    selectActor(option) {
        console.log("in select actor")
        console.log(option)
        this.stashdbUrl=option.Url.replace("https://stashdb.org/performers/","")
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
        this.stashdbUrl=option.Url.replace("https://stashdb.org/performers/","")
        console.log('/api/extref/stashdb/link2actor/' + this.actor.id +'/'+this.stashdbUrl)
        ky.get('/api/extref/stashdb/link2actor/' + this.actor.id +'/'+this.stashdbUrl ).json().then(data => {          
          // this.$store.commit('sceneList/updateScene', data)
           this.$store.commit('overlay/showActorDetails', { actor: data })
          this.close()
        })
    },    
    getImageURL (u) {        
      console.log("getimage",u)
      if (u != undefined && u.startsWith('http')) {
        return '/img/120x/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    openDialog(actor) {
      console.log("in openDialogOld")
        this.isModalActive = true
        console.log("openDialogOld isModalActive", this.isModalActive)
        this.searchStashdb()
        console.log("openDialogOld2 isModalActive", this.isModalActive)
        this.$nextTick(() => {
          console.log("nextTick", this.isModalActive)
            if (this.$refs.autocompleteInput) {
                this.$refs.autocompleteInput.focus();
            }
        });
        this.actor = actor
    },
    dump(p) {
console.log("********** dump",p)
    }
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
