/*
maxlength_indicator.js

Apply class "maxlength_indicator" to any div / element to insert the text
indicating the limit of a form element. The data-name attribute must match the
name attribute of the input being tracked.


Attributes:

  data-name(required):
    The name of the form input whose maxlength is being tracked.

  data-type(required):
    Used to indicate how the maxlength is being tracked. Current allowed
    values:

      data-type="length"
        Normal input fields that have a maxlength limit

      data-type="link"
        Tracks the maxlength of each individual link, as long as it is one link
        per line.

  data-link-length(required when data-type="link"):
    The maxlength of each individual link.


Examples:

  Normal input field:
    <form>
      <div class="maxlength_indicator"
           data-type="length"
           data-name="MyInput">
      </div>
      <textarea name="MyInput" maxlength="200"></textarea>
    </form>

  Textarea with links limited to 500 char each:
    <form>
      <div class="maxlength_indicator"
           data-type="links"
           data-name="MyLinks"
           data-link-length="500">
      </div>
      <textarea name="MyLinks"></textarea>
    </form>
*/

let limits = {};
/*
Warning class is applied to the indicator when the field's length is this many
letters or less away from the field's limit.
*/
const char_limit_warning = 10;

/* Update current limit to equal field length. */
function updateFieldLength(element) {
  const length = element.value.length;
  limits[element.name]['current'] = length;
}

/* Update the array storing the current length of each array. */
function updateLinksLength(element) {
  const links = element.value.split('\n');
  /*
  Set the array length equal to the number of links. This will remove any
  links still in the array that were deleted / are no longer present in
  the form field.
  */
  limits[element.name]['current'].length = links.length;
  for(let i = 0; i < links.length; ++i) {
    limits[element.name]['current'][i] = links[i].length;
  }
}

/* Update innerHTML of the maxlength_indicator element. */
function updateIndicator(element, type=null) {
  const {current, max} = limits[element.getAttribute('data-name')];
  if(type === 'links') {
    let values = current.slice();
    for(let i = 0; i < values.length; ++i) {
      values[i] = applyLimitClassDiv(values[i], max);
    }
    element.innerHTML = `${values.join('')}`;

  } else {
    element.innerHTML = applyLimitClassDiv(current, max);
  }
}

/*
Returns a div for each limit in the format of "current / max" and applies
CSS classes based on how close it is to the limit.
*/
function applyLimitClassDiv(current, max) {
  const remaining = max - current;
  let elClass = '';

  if(remaining <= 0) {
    elClass = 'limit';
  } else if(remaining <= char_limit_warning) {
    elClass = 'warning';
  }

  return `<div class="${elClass}">${current} / ${max}</div>`;
}

/*
Once content is loaded, find all divs with a class of maxlength_indicator and
the corresponding input element that has a name matching the data-name, and
apply event listeners to track the length of the input's content.
*/
document.addEventListener('DOMContentLoaded', function() {
  const limitEls = document.querySelectorAll('.maxlength_indicator');

  limitEls.forEach(limitEl => {
    const type = limitEl.getAttribute('data-type');
    const name = limitEl.getAttribute('data-name');

    if(type === null) {
      console.error('Maxlength indicator missing data-type on element.')
      console.error(limitEl);
    }
    if(name === null) {
      console.error('Maxlength indicator missing data-name on element.')
      console.error(limitEl);
    }

    /* Find the first form element that has a name that matches data-name. */
    const limitInput = document.querySelector(`form [name="${name}"]`);
    if(limitInput === null) console.error('Unable to find matching form element with name = ' + name);

    /* Handle regular input elements with a maxlength. */
    if(type === 'length') {
      const limit = limitInput.getAttribute('maxlength')

      if(limit === null) {
        console.error('Element missing maxlength property.');
        console.error(limitInput);
      }

      limits[name] = {
        current: 0,
        max: limit
      };

      limitInput.addEventListener('input', () => {
        updateFieldLength(limitInput);
        updateIndicator(limitEl);
      });
      /*
      Call this once to update the indicators immediately to reflect any text
      that is already in a field, such as when a page is refreshed and the
      input still contains text.
      */
      updateFieldLength(limitInput);
      updateIndicator(limitEl);

    /*
    Handle input elements that limit the lengths of indiviual lines (such as
    links).
    */
    } else if(type === 'link'){
      const limit = limitEl.getAttribute('data-link-length')

      if(limit === null) {
        console.error('Field limit indicator missing data-link-length property.');
        console.error(limitInput);
      }

      limits[name] = {
        current: [],
        max: limit
      };

      limitInput.addEventListener('input', () => {
        updateLinksLength(limitInput);
        updateIndicator(limitEl, 'links');
      });
      /*
      Call this once to update the indicators immediately to reflect any text
      that is already in a field, such as when a page is refreshed and the
      input still contains text.
      */
      updateLinksLength(limitInput);
      updateIndicator(limitEl, 'links');
    }
  });
});
