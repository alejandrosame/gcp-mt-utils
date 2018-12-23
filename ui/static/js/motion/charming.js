!function(element){
    "undefined"==typeof module?this.charming=element:module.exports=element
}(function (element, options) {
  options = options || {}
  element.normalize()
  var splitRegex = options.splitRegex

  var tagName = options.tagName || 'span'
  var classPrefix = options.classPrefix != null ? options.classPrefix : 'char'
  var count = 1

  function inject (element) {
    var parentNode = element.parentNode
    var string = element.nodeValue
    var split = splitRegex ? string.split(splitRegex) : string
    var length = split.length
    var word = ""
    var nWords = 0
    var i = -1
    while (++i < length) {
      var node = document.createElement(tagName)
      if (classPrefix) {
        node.className = classPrefix + count
        count++
      }
      if (split[i] == "\n"){
        node.appendChild(document.createTextNode(word))
        node.appendChild(document.createElement("br"))
        word = ""
        nWords = 0
      }
      else if (split[i] == " "){
        word += split[i]
        nWords += 1

        if (nWords == 10){
            node.appendChild(document.createTextNode(word))
            word = ""
            nWords = 0
        }

      }
      else {
        word += split[i]
      }
      node.setAttribute('aria-hidden', 'true')
      parentNode.insertBefore(node, element)
    }
    if (word != ""){
        node.appendChild(document.createTextNode(word))
    }
    if (string.trim() !== '') {
      parentNode.setAttribute('aria-label', string)
    }
    parentNode.removeChild(element)
  }

  ;(function traverse (element) {
    // `element` is itself a text node.
    if (element.nodeType === 3) {
      return inject(element)
    }

    // `element` has a single child text node.
    var childNodes = Array.prototype.slice.call(element.childNodes) // static array of nodes
    var length = childNodes.length
    if (length === 1 && childNodes[0].nodeType === 3) {
      return inject(childNodes[0])
    }

    // `element` has more than one child node.
    var i = -1
    while (++i < length) {
      traverse(childNodes[i])
    }
  })(element)
});