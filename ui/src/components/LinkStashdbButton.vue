<template>
    <span>
        <a :class="buttonClass"
            @click="openDialog(item)"
            :title="'Link to stashdb'">
            <b-icon pack="mdi" :icon="'link-variant-plus'" size="is-small"/>
        </a>
    </span>
</template>

<script>
export default {
  name: 'LinkStashdbButton',
  props: { item: Object },  
  data () {
    return {
        isModalActive: false,
        searchResults: [],
        query: "",
        queryString: "",
        isFetching: false,
        }
  },
  computed: {
    buttonClass () {
      return 'button  is-outlined is-small'
    }
  },
  methods: {
    openDialog(scene) {
        console.log("stash button open")
      this.$store.commit('overlay/showSearchStashdb', { scene })
    },
     }
}
</script>
