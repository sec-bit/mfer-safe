export default function Fieldset(props) {
  const { legend, children } = props;
  return (
    <details open>
      <summary>
        <span className="open">
          {legend} {"▾"}
        </span>
      </summary>
      <fieldset>
        <legend>
          {legend} <span className="close">{"▴"}</span>
        </legend>
        {children}
      </fieldset>
    </details>
  );
}
